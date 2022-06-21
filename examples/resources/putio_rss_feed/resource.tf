resource "putio_rss_feed" "example" {
  title          = "Test RSS feed"
  rss_source_url = "http://example.com/feed.rss"
  keyword        = ""

  delete_old_files        = false
  dont_process_whole_feed = false
  parent_dir_id           = 0
  paused                  = false
  unwanted_keywords       = ""
}
