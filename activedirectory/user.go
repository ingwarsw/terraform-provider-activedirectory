package activedirectory

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/ldap.v3"
)

// User is the base implementation of ad User object
type User struct {
	dn          string
	attributes  map[string]string
}

// returns User object
func (api *API) getUser(firstName, lastName string) (*User, error) {
	log.Infof("Searching ad User '%s %s'", firstName, lastName)

	domain := api.getDomainDN()

	// ldap filter
	filter := fmt.Sprintf("(&(objectclass=User)(name=%s %s))", firstName, lastName)

	// trying to get ou object
	ret, err := api.searchObject(filter, domain, []string{})
	if err != nil {
		return nil, fmt.Errorf("getUser - searching for User object '%s %s' failed: %s", firstName, lastName, err)
	}

	if len(ret) == 0 {
		return nil, nil
	}

	if len(ret) > 1 {
		return nil, fmt.Errorf("getUser - more than one User object with the same name found")
	}

	return &User{
		dn:          ret[0].dn,
		attributes:  simplifyAttributes(ret[0].attributes),
	}, nil
}

// creates a new User object
func (api *API) createUser(dn string, attributes map[string]string) error {
	log.Infof("Creating User object %s", dn)

	//tmp, err := api.getUser(firstName, lastName)
	//if err != nil {
	//	return fmt.Errorf("createUser - talking to active directory failed: %s", err)
	//}

	// there is already a User object with the same name
	//if tmp != nil {
		//if tmp.name == firstName && tmp.dn == fmt.Sprintf("cn=%s,%s", firstName, ou) {
		//	log.Infof("User object %s already exists, updating description", firstName)
		//	return api.updateUserDescription(firstName, ou, description)
		//}

		//return fmt.Errorf("createUser - User object %s already exists in a different ou", firstName)
	//}

	//attributes := make(map[string][]string)
	//attributes["name"] = []string{firstName + " " + lastName}
	//attributes["GivenName"] = []string{firstName}
	//attributes["sAMAccountName"] = []string{firstName}
	//attributes["sn"] = []string{lastName}
	//attributes["userAccountControl"] = []string{"544"}
	//attributes["description"] = []string{description}
	return api.createObject(dn, []string{"User"}, mapAttributes(attributes))
}

// moves an existing User object to a new ou
func (api *API) updateUserOU(cn, ou, newOU string) error {
	log.Infof("Moving User object %s from %s to %s", cn, ou, newOU)

	tmp, err := api.getUser(cn, cn)
	if err != nil {
		return fmt.Errorf("updateUserOU - talking to active directory failed: %s", err)
	}

	if tmp == nil {
		return fmt.Errorf("updateUserOU - User object %s does not exists: %s", cn, err)
	}

	// User object is already in the target OU, nothing to do
	if strings.EqualFold(tmp.dn, fmt.Sprintf("cn=%s,%s", cn, newOU)) {
		log.Infof("User object is already in the target ou")
		return nil
	}

	// specific uid of the User
	UserUID := fmt.Sprintf("cn=%s", cn)

	// move User object to new ou
	req := ldap.NewModifyDNRequest(fmt.Sprintf("cn=%s,%s", cn, ou), UserUID, true, newOU)
	if err := api.client.ModifyDN(req); err != nil {
		return fmt.Errorf("updateUserOU - failed to move User object: %s", err)
	}

	log.Info("Object moved successfully")
	return nil
}

// updates the description of an existing User object
func (api *API) updateUserDescription(cn, ou, description string) error {
	log.Infof("Updating description of User object %s", cn)
	return api.updateObject(fmt.Sprintf("cn=%s,%s", cn, ou), nil, nil, map[string][]string{
		"description": {description},
	}, nil)
}

// deletes an existing User object.
func (api *API) deleteUser(dn string) error {
	log.Infof("Deleting User object: %s", dn)
	return api.deleteObject(dn)
}

func getValue(attribute []string) string {
	if len(attribute) > 0 {
		return attribute[0]
	}
	return ""
}

func mapAttributes(attributes map[string]string) map[string][]string {
	adAttributes := make(map[string][]string)
	for key, value := range attributes {
		adAttributes[key] = []string{value}
	}
	return adAttributes
}

func simplifyAttributes(adAttributes map[string][]string) map[string]string {
	attributes := make(map[string]string)
	for key, value := range adAttributes {
		attributes[key] = value[0]
	}
	return attributes
}