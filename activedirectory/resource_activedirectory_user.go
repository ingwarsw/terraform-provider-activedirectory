package activedirectory

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	log "github.com/sirupsen/logrus"
)

// resourceADOUserObject is the main function for ad user terraform resource
func resourceADOUserObject() *schema.Resource {
	return &schema.Resource{
		Create: resourceADOUserObjectCreate,
		Read:   resourceADOUserObjectRead,
		Update: resourceADOUserObjectUpdate,
		Delete: resourceADOUserObjectDelete,

		Schema: map[string]*schema.Schema{
			"first_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// this is to ignore case in ad distinguished name
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.EqualFold(old, new)
				},
			},
			"last_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// this is to ignore case in ad distinguished name
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.EqualFold(old, new)
				},
			},
			"ou": {
				Type:     schema.TypeString,
				Required: true,
				// this is to ignore case in ad distinguished name
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.EqualFold(old, new)
				},
				StateFunc: func(val interface{}) string {
					return strings.ToLower(val.(string))
				},
			},
			"login": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// this is to ignore case in ad distinguished name
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.EqualFold(old, new)
				},
			},
			"email": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// this is to ignore case in ad distinguished name
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.EqualFold(old, new)
				},
			},
			"password": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Sensitive: true,
				// this is to ignore case in ad distinguished name
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.EqualFold(old, new)
				},
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  nil,
			},
		},
	}
}

// resourceADOUserObjectCreate is 'create' part of terraform CRUD functions for AD provider
func resourceADOUserObjectCreate(d *schema.ResourceData, meta interface{}) error {
	log.Infof("Creating AD user object")

	api := meta.(APIInterface)

	if err := api.createUser(d.Get("first_name").(string),
		d.Get("last_name").(string),
		d.Get("ou").(string),
		d.Get("description").(string)); err != nil {
		return fmt.Errorf("resourceADOUserObjectCreate - create - %s", err)
	}

	d.SetId(strings.ToLower(fmt.Sprintf("cn=%s %s,%s", d.Get("first_name").(string), d.Get("last_name").(string), d.Get("ou").(string))))
	return resourceADOUserObjectRead(d, meta)
}

// resourceADOUserObjectRead is 'read' part of terraform CRUD functions for AD provider
func resourceADOUserObjectRead(d *schema.ResourceData, meta interface{}) error {
	log.Infof("Reading AD user object")

	api := meta.(APIInterface)

	User, err := api.getUser(d.Get("first_name").(string), d.Get("last_name").(string))
	if err != nil {
		return fmt.Errorf("resourceADOUserObjectRead - getUser - %s", err)
	}

	if User == nil {
		log.Infof("User object %s no longer exists", d.Get("name").(string))

		d.SetId("")
		return nil
	}

	d.SetId(strings.ToLower(User.dn))

	if err := d.Set("description", User.description); err != nil {
		return fmt.Errorf("resourceADOUserObjectRead - set description - failed to set description: %s", err)
	}

	return nil
}

// resourceADOUserObjectUpdate is 'update' part of terraform CRUD functions for ad provider
func resourceADOUserObjectUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Infof("Updating AD User object")

	api := meta.(APIInterface)

	oldOU, newOU := d.GetChange("ou")

	// let's try to update in parts
	d.Partial(true)

	// check description
	if d.HasChange("description") {
		if err := api.updateUserDescription(d.Get("name").(string), oldOU.(string), d.Get("description").(string)); err != nil {
			return fmt.Errorf("resourceADOUserObjectUpdate - update description - %s", err)
		}

		d.SetPartial("description")
	}

	// check ou
	if d.HasChange("ou") {
		if err := api.updateUserOU(d.Get("name").(string), oldOU.(string), newOU.(string)); err != nil {
			return fmt.Errorf("resourceADOUserObjectUpdate - update ou - %s", err)
		}
	}

	d.Partial(false)
	d.SetId(strings.ToLower(fmt.Sprintf("cn=%s,%s", d.Get("name").(string), d.Get("ou").(string))))

	// read current ad data to avoid drift
	return resourceADOUserObjectRead(d, meta)
}

// resourceADOUserObjectDelete is 'delete' part of terraform CRUD functions for ad provider
func resourceADOUserObjectDelete(d *schema.ResourceData, meta interface{}) error {
	log.Infof("Deleting AD User object")

	api := meta.(APIInterface)

	// call ad to delete the User object, no error means that object was deleted successfully
	return api.deleteUser(d.Get("name").(string), "", d.Get("ou").(string))
}
