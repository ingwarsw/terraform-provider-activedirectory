package activedirectory

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	log "github.com/sirupsen/logrus"
)

// resourceADGroupObject is the main function for ad ou terraform resource
func resourceADGroupObject() *schema.Resource {
	return &schema.Resource{
		Create: resourceADGroupObjectUpdate,
		Read:   resourceADGroupObjectRead,
		Update: resourceADGroupObjectUpdate,
		Delete: resourceADGroupObjectDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				// this is to ignore case in ad distinguished name
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.EqualFold(old, new)
				},
			},
			"base_ou": {
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
			"user_base": {
				Type:     schema.TypeString,
				Optional:    true,
				Default:     "",
				// this is to ignore case in ad distinguished name
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.EqualFold(old, new)
				},
				StateFunc: func(val interface{}) string {
					return strings.ToLower(val.(string))
				},
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  nil,
			},
			"ignore_members_unknown_by_terraform": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Ignore members which are unknown by terraform",
			},
			"member": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				Default:  nil,
			},
		},
	}
}

// resourceADGroupObjectCreate is 'create' part of terraform CRUD functions for AD provider
func resourceADGroupObjectCreate(d *schema.ResourceData, meta interface{}) error {
	log.Infof("Creating AD Group object")
	api := meta.(APIInterface)

	members := make([]string, 0)
	for _, m := range d.Get("member").(*schema.Set).List() {
		if m.(string) != "" {
			members = append(members, m.(string))
		}
	}
	log.Infof("Member count from config %d", len(members))
	if err := api.createGroup(d.Get("name").(string), d.Get("base_ou").(string),
		d.Get("description").(string), d.Get("user_base").(string), members, d.Get("ignore_members_unknown_by_terraform").(bool)); err != nil {
		return fmt.Errorf("resourceADGroupObjectCreate - create ou - %s", err)
	}

	d.SetId(strings.ToLower(fmt.Sprintf("ou=%s,%s", d.Get("name").(string), d.Get("base_ou").(string))))
	return resourceADGroupObjectRead(d, meta)
}

// resourceADGroupObjectRead is 'read' part of terraform CRUD functions for AD provider
func resourceADGroupObjectRead(d *schema.ResourceData, meta interface{}) error {
	log.Infof("Reading AD Group object")

	api := meta.(APIInterface)
	membersFromHCL := d.Get("member").(*schema.Set).List()
	members := make([]string, len(membersFromHCL))
	for i, m := range membersFromHCL {
		members[i] = m.(string)
	}
	log.Infof("resourceADGroupObjectRead - members from hcl %s", members)

	group, err := api.getGroup(d.Get("name").(string),
		d.Get("base_ou").(string),
		d.Get("user_base").(string),
		members,
		d.Get("ignore_members_unknown_by_terraform").(bool))
	if err != nil {
		return fmt.Errorf("resourceADGroupObjectRead - get group - %s", err)
	}

	if group == nil {
		log.Infof("Group object %s no longer exists under %s", d.Get("name").(string), d.Get("base_ou").(string))

		d.SetId("")
		return nil
	}

	if err := d.Set("name", group.name); err != nil {
		return fmt.Errorf("resourceADGroupObjectRead - set name - failed to set group name to %s: %s", group.name, err)
	}

	baseOU := strings.ToLower(group.dn[(len(group.name) + 1 + 3):]) // remove 'group=' and ',' and group name
	if err := d.Set("base_ou", baseOU); err != nil {
		return fmt.Errorf("resourceADGroupObjectRead - set base_ou - failed to set group base_ou to %s: %s", baseOU, err)
	}

	if err := d.Set("description", group.description); err != nil {
		return fmt.Errorf("resourceADGroupObjectRead - set description - failed to set group description to %s: %s", group.description, err)
	}
	if err := d.Set("member", group.member); err != nil {
		return fmt.Errorf("resourceADGroupObjectRead - set member - failed to set group member to %s: %s", group.member, err)
	}

	d.SetId(strings.ToLower(group.dn))

	return nil
}

// resourceADGroupObjectUpdate is 'update' part of terraform CRUD functions for ad provider
func resourceADGroupObjectUpdate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(strings.ToLower(fmt.Sprintf("ou=%s,%s", d.Get("name").(string), d.Get("base_ou").(string))))
	return resourceADGroupObjectChange(d, meta, true)
}

func resourceADGroupObjectChange(d *schema.ResourceData, meta interface{}, add bool) error {
	log.Infof("Updating AD Group object")

	api := meta.(APIInterface)

	oldOU := d.Get("base_ou").(string)
	oldName := d.Get("name").(string)
	oldUserBase := d.Get("user_base").(string)

	member := d.Get("member")
	memberList := make([]string, 0)
	for _, m := range member.(*schema.Set).List() {
		memberList = append(memberList, m.(string))
	}

	oldList := make([]string, 0)
	newList := make([]string, 0)

	if add {
		newList = memberList
	} else {
		oldList = memberList
	}

	log.Infof("Old members %s, New members %s", oldList, newList)
	ignoreMembersUnknownByTerraform := d.Get("ignore_members_unknown_by_terraform").(bool)
	if err := api.updateGroupMembers(
		oldName,
		oldOU,
		oldUserBase,
		oldList,
		newList,
		ignoreMembersUnknownByTerraform); err != nil {
		return fmt.Errorf("resourceADGroupObjectUpdate - update members - %s", err)
	}

	return resourceADGroupObjectRead(d, meta)
}

// resourceADGroupObjectDelete is 'delete' part of terraform CRUD functions for ad provider
func resourceADGroupObjectDelete(d *schema.ResourceData, meta interface{}) error {
	log.Infof("Deleting AD Group object")
	return resourceADGroupObjectChange(d, meta, false)
}
