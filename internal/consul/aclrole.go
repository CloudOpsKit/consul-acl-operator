package consul

import (
	operatorConfig "github.com/CloudOpsKit/consul-acl-operator/internal/config"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

func CreateOrUpdateAclRole(reqLogger *zap.SugaredLogger, config operatorConfig.Config, role *api.ACLRole) (roleID string, err error) {
	client, err := api.NewClient(&config.ConsulConfig)
	if err != nil {
		reqLogger.Error(err, "Unable to create client Object")
		return "", err
	}

	if role.ID == "" {
		reqLogger.Warn("AclRole with name " + role.Name + " does not have ID field")
		r, _, _ := client.ACL().RoleReadByName(role.Name, nil)
		if r != nil {
			reqLogger.Debug("Found AclRole")
			role.ID = r.ID
		} else {
			r, _, err = client.ACL().RoleCreate(role, nil)
			if err != nil {
				reqLogger.Error(err, "Unable to create AclRole")
				return "", err
			}
			reqLogger.Debug("Successfully created AclRole")
			return r.ID, err
		}
	}
	r, _, err := client.ACL().RoleRead(role.ID, nil)
	if err != nil {
		reqLogger.Error(err, "Can not read AclRole")
		return "", err
	} else if r == nil {
		role.ID = ""
		r, _, err = client.ACL().RoleCreate(role, nil)
		if err != nil {
			reqLogger.Error(err, "Unable to create AclRole")
			return "", err
		}
		reqLogger.Debug("Successfully created AclRole")
		return r.ID, err
	}

	_, _, err = client.ACL().RoleUpdate(role, nil)
	if err != nil {
		reqLogger.Error(err, "Unable to update AclRole")
		return "", err
	}
	reqLogger.Debug("Successfully updated AclRole")

	return r.ID, err
}

func DeleteAclRole(reqLogger *zap.SugaredLogger, config operatorConfig.Config, roleName string) (err error) {
	client, err := api.NewClient(&config.ConsulConfig)
	if err != nil {
		reqLogger.Error(err, "Unable to create client Object")
		return err
	}

	r, _, err := client.ACL().RoleReadByName(roleName, nil)
	if r == nil {
		reqLogger.Error(err, "Can not find AclRole")
		return nil
	}
	roleID := r.ID

	if _, err := client.ACL().RoleDelete(roleID, nil); err != nil {
		reqLogger.Error(err, "Unable to remove AclRole")
		return err
	}

	reqLogger.Debug("Successfully removed AclRole")
	return err
}
