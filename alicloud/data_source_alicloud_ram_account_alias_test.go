package alicloud

import (
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccAlicloudAccountAliasDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAlicloudAccountAliasDataSourceBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_ram_account_alias.alias"),
					resource.TestCheckResourceAttr("data.alicloud_ram_account_alias.alias", "account_alias", "1307087942598154"),
				),
			},
		},
	})
}

const testAccCheckAlicloudAccountAliasDataSourceBasic = `
data "alicloud_ram_account_alias" "alias" {
}`
