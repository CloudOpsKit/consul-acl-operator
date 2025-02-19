package consul

import (
	operatorConfig "github.com/CloudOpsKit/consul-acl-operator/internal/config"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

func CreateOrUpdateAuthMethod(reqLogger *zap.SugaredLogger, config operatorConfig.Config, authMethod *api.ACLAuthMethod) (err error) {

	client, err := api.NewClient(&config.ConsulConfig)
	if err != nil {
		reqLogger.Error(err, "Unable to create client Object")
		return err
	}

	r, _, _ := client.ACL().AuthMethodRead(authMethod.Name, nil)
	if r != nil {
		reqLogger.Debug("Found AuthMethod with Name: " + r.Name)
	} else {
		r, _, err = client.ACL().AuthMethodCreate(authMethod, nil)
		if err != nil {
			reqLogger.Error("Unable to create AuthMethod")
			return err
		}
		reqLogger.Debug("Successfully created AuthMethod")
		return err
	}
	_, _, err = client.ACL().AuthMethodUpdate(authMethod, nil)
	if err != nil {
		reqLogger.Error("Unable to update AuthMethod")
		return err
	}
	reqLogger.Debug("Successfully updated AuthMethod")

	return err
}

func DeleteAuthMethod(reqLogger *zap.SugaredLogger, config operatorConfig.Config, authMethodName string) (err error) {

	client, err := api.NewClient(&config.ConsulConfig)
	if err != nil {
		reqLogger.Error(err, "Unable to create client Object")
		return err
	}

	m, _, err := client.ACL().AuthMethodRead(authMethodName, nil)
	if m == nil {
		reqLogger.Error(err, "Can not find AuthMethod")
		return nil
	}

	_, err = client.ACL().AuthMethodDelete(authMethodName, nil)
	if err != nil {
		reqLogger.Error(err, "Can not delete AuthMethod")
		return nil
	}

	reqLogger.Debug("Successfully removed AuthMethod")
	return err
}
