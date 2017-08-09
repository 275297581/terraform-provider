package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/ram"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"time"
)

func resourceAlicloudRamGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAlicloudRamGroupCreate,
		Read:   resourceAlicloudRamGroupRead,
		Update: resourceAlicloudRamGroupUpdate,
		Delete: resourceAlicloudRamGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"group_name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRamGroupName,
			},
			"comments": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateComment,
			},
			"force": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceAlicloudRamGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ramconn

	args := ram.GroupRequest{
		Group: ram.Group{
			GroupName: d.Get("group_name").(string),
		},
	}

	response, err := conn.CreateGroup(args)
	if err != nil {
		return fmt.Errorf("CreateGroup got an error: %#v", err)
	}

	d.SetId(response.Group.GroupName)
	return resourceAlicloudRamGroupUpdate(d, meta)
}

func resourceAlicloudRamGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ramconn

	d.Partial(true)

	args := ram.GroupUpdateRequest{
		GroupName: d.Id(),
	}
	attributeUpdate := false

	if d.HasChange("group_name") && !d.IsNewResource() {
		ov, nv := d.GetChange("group_name")
		args.GroupName = ov.(string)
		args.NewGroupName = nv.(string)
		d.SetPartial("group_name")
		attributeUpdate = true
	}

	if d.HasChange("comments") {
		d.SetPartial("comments")
		args.NewComments = d.Get("comments").(string)
		attributeUpdate = true
	}

	if attributeUpdate {
		if _, err := conn.UpdateGroup(args); err != nil {
			return fmt.Errorf("UpdateGroup got an error: %v", err)
		}
	}

	d.Partial(false)
	return resourceAlicloudRamGroupRead(d, meta)
}

func resourceAlicloudRamGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ramconn

	args := ram.GroupQueryRequest{
		GroupName: d.Id(),
	}

	response, err := conn.GetGroup(args)
	if err != nil {
		if RamEntityNotExist(err) {
			d.SetId("")
		}
		return fmt.Errorf("GetGroup got an error: %#v", err)
	}

	group := response.Group
	d.Set("group_name", group.GroupName)
	d.Set("comments", group.Comments)
	return nil
}

func resourceAlicloudRamGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ramconn

	args := ram.GroupQueryRequest{
		GroupName: d.Id(),
	}

	if d.Get("force").(bool) {
		// list and delete users which in this group
		listUserResp, err := conn.ListUsersForGroup(args)
		if err != nil {
			return fmt.Errorf("Error while listing users for group %s: %#v", d.Id(), err)
		}
		users := listUserResp.Users.User
		if len(users) > 0 {
			for _, v := range users {
				_, err = conn.RemoveUserFromGroup(ram.UserRelateGroupRequest{
					UserName:  v.UserName,
					GroupName: args.GroupName,
				})
				if err != nil && !RamEntityNotExist(err) {
					return fmt.Errorf("Error while deleting user %s from group %s: %#v", v.UserName, d.Id(), err)
				}
			}
		}

		// list and detach policies which attach this group
		listPolicyResp, err := conn.ListPoliciesForGroup(args)
		if err != nil {
			return fmt.Errorf("Error while listing policies for group %s: %#v", d.Id(), err)
		}
		policies := listPolicyResp.Policies.Policy
		if len(policies) > 0 {
			for _, v := range policies {
				_, err = conn.DetachPolicyFromGroup(ram.AttachPolicyToGroupRequest{
					PolicyRequest: ram.PolicyRequest{
						PolicyType: v.PolicyType,
						PolicyName: v.PolicyName,
					},
					GroupName: args.GroupName,
				})
				if err != nil && !RamEntityNotExist(err) {
					return fmt.Errorf("Error while detaching policy %s from group %s: %#v", v.PolicyName, d.Id(), err)
				}
			}
		}
	}

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		if _, err := conn.DeleteGroup(args); err != nil {
			if IsExceptedError(err, DeleteConflictGroupUser) || IsExceptedError(err, DeleteConflictGroupPolicy) {
				return resource.RetryableError(fmt.Errorf("The group can not has any user member or any attached policy while deleting the group.- you can set force with true to force delete the group."))
			}
			return resource.NonRetryableError(fmt.Errorf("Error deleting group %s: %#v, you can set force with true to force delete the group.", d.Id(), err))
		}
		return nil
	})
}
