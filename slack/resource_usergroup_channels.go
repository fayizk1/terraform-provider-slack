package slack

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/slack-go/slack"
	"log"
	"strings"
)

func resourceSlackUserGroupChannels() *schema.Resource {
	return &schema.Resource{
		Read:   resourceSlackUserGroupChannelsRead,
		Create: resourceSlackUserGroupChannelsCreate,
		Update: resourceSlackUserGroupChannelsUpdate,
		Delete: resourceSlackUserGroupChannelsDelete,

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
			"channels": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required: true,
			},
		},
	}
}

func configureSlackUserGroupChannels(d *schema.ResourceData, userGroup slack.UserGroup) {
	d.SetId(userGroup.ID)
	_ = d.Set("channels", userGroup.Prefs.Channels)
}

func resourceSlackUserGroupChannelsCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Team).client

	ctx := context.WithValue(context.Background(), ctxId, d.Id())

	usergroupId := d.Get("usergroup_id").(string)
	log.Printf("[DEBUG] Creating usergroup channels relation: %s", usergroupId)

	iChannels := d.Get("channels").([]interface{})
	channelsIds := make([]string, len(iChannels))
	for i, v := range iChannels {
		channelsIds[i] = v.(string)
	}

	params := &slack.UserGroup{
		ID: usergroupId,
		Prefs: slack.UserGroupPrefs{
			Channels: channelsIds,
		},
	}

	userGroup, err := client.UpdateUserGroupContext(ctx, *params)

	if err != nil {
		return fmt.Errorf("user group channel create error: %s (%v),  %s", usergroupId, channelsIds, err.Error())
	}

	configureSlackUserGroupChannels(d, userGroup)

	return nil
}

func resourceSlackUserGroupChannelsRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Team).client

	ctx := context.WithValue(context.Background(), ctxId, d.Id())

	usergroupId := d.Get("usergroup_id").(string)
	log.Printf("[DEBUG] Reading usergroup channels relation: %s", usergroupId)

	if usergroupId != d.Id() {
		return fmt.Errorf("it looks usergroup id has been changed but it's not allowed. Res ID: %s", d.Id())
	}

	// Use a cache for usergroups api call because the limitation is strict
	var userGroups *[]slack.UserGroup

	if !restoreJsonCache(userGroupListCacheFileName, &userGroups) {
		tempUserGroups, err := client.GetUserGroupsContext(ctx, func(params *slack.GetUserGroupsParams) {
			params.IncludeUsers = false
			params.IncludeCount = false
			params.IncludeDisabled = true
		})

		if err != nil {
			return fmt.Errorf("user group channel read error: %s,  %s", usergroupId, err.Error())
		}

		userGroups = &tempUserGroups

		saveCacheAsJson(userGroupListCacheFileName, &userGroups)
	}

	if userGroups == nil {
		panic(fmt.Errorf("a serious error happened. please create an issue to https://github.com/jmatsu/terraform-provider-slack"))
	}

	for _, userGroup := range *userGroups {
		if userGroup.ID == usergroupId {
			configureSlackUserGroupChannels(d, userGroup)
			return nil
		}
	}

	return nil
}

func resourceSlackUserGroupChannelsUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Team).client

	ctx := context.WithValue(context.Background(), ctxId, d.Id())

	usergroupId := d.Get("usergroup_id").(string)
	log.Printf("[DEBUG] Updating usergroup channels relation: %s", usergroupId)

	if usergroupId != d.Id() {
		return fmt.Errorf("it looks usergroup id has been changed but it's not allowed. Res ID: %s", d.Id())
	}

	iChannels := d.Get("channels").([]interface{})
	channelsIds := make([]string, len(iChannels))
	for i, v := range iChannels {
		channelsIds[i] = v.(string)
	}

	params := &slack.UserGroup{
		ID: usergroupId,
		Prefs: slack.UserGroupPrefs{
			Channels: channelsIds,
		},
	}

	userGroup, err := client.UpdateUserGroupContext(ctx, *params)

	if err != nil {
		return fmt.Errorf("user group channel update error: %s (%v),  %s", usergroupId, channelsIds, err.Error())
	}

	configureSlackUserGroupChannels(d, userGroup)

	return nil
}

func resourceSlackUserGroupChannelsDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Team).client

	ctx := context.WithValue(context.Background(), ctxId, d.Id())
	usergroupId := d.Get("usergroup_id").(string)

	log.Printf("[DEBUG] Deleting usergroup channels relation: %s", usergroupId)

	if usergroupId != d.Id() {
		return fmt.Errorf("it looks usergroup id has been changed but it's not allowed. Res ID: %s", d.Id())
	}

	params := &slack.UserGroup{
		ID: usergroupId,
		Prefs: slack.UserGroupPrefs{
			Channels: []string{},
		},
	}

	_, err := client.UpdateUserGroupContext(ctx, *params)
	if err != nil && !strings.Contains(err.Error(), "no_such_subteam") {
		return fmt.Errorf("user group channel update error: %s,  %s", usergroupId, err.Error())
	}

	d.SetId("")
	return nil
}
