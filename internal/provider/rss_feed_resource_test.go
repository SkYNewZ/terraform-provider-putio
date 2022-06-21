package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccRssFeedResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRssFeedConfig("one", "https://google.fr", "foo"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("putio_rss_feed.test", "id"),
					resource.TestCheckResourceAttr("putio_rss_feed.test", "title", "one"),
					resource.TestCheckResourceAttr("putio_rss_feed.test", "rss_source_url", "https://google.fr"),
					resource.TestCheckResourceAttr("putio_rss_feed.test", "parent_dir_id", "0"),
					resource.TestCheckResourceAttr("putio_rss_feed.test", "delete_old_files", "false"),
					// resource.TestCheckResourceAttr("putio_rss_feed.test", "dont_process_whole_feed", "false"),
					resource.TestCheckResourceAttr("putio_rss_feed.test", "keyword", "foo"),
					resource.TestCheckResourceAttr("putio_rss_feed.test", "unwanted_keywords", ""),
					resource.TestCheckResourceAttr("putio_rss_feed.test", "paused", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "putio_rss_feed.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccRssFeedConfig("two", "https://google.fr", "foo"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("putio_rss_feed.test", "id"),
					resource.TestCheckResourceAttr("putio_rss_feed.test", "title", "two"),
					resource.TestCheckResourceAttr("putio_rss_feed.test", "rss_source_url", "https://google.fr"),
					resource.TestCheckResourceAttr("putio_rss_feed.test", "parent_dir_id", "0"),
					resource.TestCheckResourceAttr("putio_rss_feed.test", "delete_old_files", "false"),
					// resource.TestCheckResourceAttr("putio_rss_feed.test", "dont_process_whole_feed", "false"),
					resource.TestCheckResourceAttr("putio_rss_feed.test", "keyword", "foo"),
					resource.TestCheckResourceAttr("putio_rss_feed.test", "unwanted_keywords", ""),
					resource.TestCheckResourceAttr("putio_rss_feed.test", "paused", "false"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccRssFeedConfig(title, url, keyword string) string {
	return fmt.Sprintf(`resource "putio_rss_feed" "test" {
  title = %q
  rss_source_url = %q
  keyword = %q
}
`, title, url, keyword)
}
