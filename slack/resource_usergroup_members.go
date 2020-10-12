package slack

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/slack-go/slack"
	"log"
	"strings"
)

func resourceSlackUserGroupMembers() *schema.Resource {
	return &schema.Resource{
		Read:   resourceSlackUserGroupMembersRead,
		Create: resourceSlackUserGroupMembersCreate,
		Update: resourceSlackUserGroupMembersUpdate,
		Delete: resourceSlackUserGroupMembersDelete,

		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
				_ = d.Set("usergroup_id", d.Id())
				return schema.ImportStatePassthrough(d, m)
			},
		},

		Schema: map[string]*schema.Schema{
			"usergroup_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"members": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required: true,
			},
		},
	}
}

func configureSlackUserGroupMembers(d *schema.ResourceData, userGroup slack.UserGroup) {
	d.SetId(userGroup.ID)
	_ = d.Set("members", userGroup.Users)
}

func resourceSlackUserGroupMembersCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Team).client

	ctx := context.WithValue(context.Background(), ctxId, d.Id())

	usergroupId := d.Get("usergroup_id").(string)

	iMembers := d.Get("members").([]interface{})
	userIds := make([]string, len(iMembers))
	for i, v := range iMembers {
		userIds[i] = v.(string)
	}
	userIdParam := strings.Join(userIds, ",")

	log.Printf("[DEBUG] Creating usergroup members: %s (%s)", usergroupId, userIdParam)

	userGroup, err := client.UpdateUserGroupMembersContext(ctx, usergroupId, userIdParam)

	if err != nil {
		return fmt.Errorf("user group member write error: %s ,  %s", usergroupId, err.Error())
	}

	configureSlackUserGroupMembers(d, userGroup)

	return nil
}

func resourceSlackUserGroupMembersRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Team).client

	ctx := context.WithValue(context.Background(), ctxId, d.Id())

	usergroupId := d.Get("usergroup_id").(string)

	log.Printf("[DEBUG] Reading usergroup members: %s", usergroupId)

	if usergroupId != d.Id() {
		return fmt.Errorf("it looks usergroup id has been changed but it's not allowed. Res ID: %s", d.Id())
	}

	members, err := client.GetUserGroupMembersContext(ctx, usergroupId)

	if err != nil {
		if strings.Contains(err.Error(), `no_such_subteam`) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("user group member read error: %s ,  %s", usergroupId, err.Error())
	}

	_ = d.Set("members", members)

	return nil
}

func resourceSlackUserGroupMembersUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Team).client

	ctx := context.WithValue(context.Background(), ctxId, d.Id())

	usergroupId := d.Get("usergroup_id").(string)

	if usergroupId != d.Id() {
		return fmt.Errorf("it looks usergroup id has been changed but it's not allowed. Res ID: %s", d.Id())
	}

	_, err := client.EnableUserGroupContext(ctx, usergroupId)

	if err != nil && err.Error() != "already_enabled" {
		return fmt.Errorf("resource memeber update error: %s ,  %s", usergroupId, err.Error())
	}

	iMembers := d.Get("members").([]interface{})
	userIds := make([]string, len(iMembers))
	for i, v := range iMembers {
		userIds[i] = v.(string)
	}
	userIdParam := strings.Join(userIds, ",")

	log.Printf("[DEBUG] Updating usergroup members: %s (%s)", usergroupId, userIdParam)

	userGroup, err := client.UpdateUserGroupMembersContext(ctx, usergroupId, userIdParam)

	if err != nil {
		return fmt.Errorf("user group member update error: %s ,  %s", usergroupId, err.Error())
	}

	configureSlackUserGroupMembers(d, userGroup)

	return nil
}

func resourceSlackUserGroupMembersDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Team).client

	ctx := context.WithValue(context.Background(), ctxId, d.Id())
	usergroupId := d.Get("usergroup_id").(string)

	if usergroupId != d.Id() {
		return fmt.Errorf("it looks usergroup id has been changed but it's not allowed. Res ID: %s", d.Id())
	}

	log.Printf("[DEBUG] Reading usergroup members: %s", usergroupId)

	// Cannot use "" as a member parameter, so let me disable it
	if _, err := client.DisableUserGroupContext(ctx, usergroupId); err != nil && ! strings.Contains(err.Error(),
		`no_such_subteam`) {
		return fmt.Errorf("user group member delete error: %s ,  %s", usergroupId, err.Error())
	}

	d.SetId("")

	return nil
}
