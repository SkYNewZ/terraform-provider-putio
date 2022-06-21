package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/skynewz/terraform-provider-putio/internal/modifiers"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ tfsdk.ResourceType = rssFeedResourceType{}
var _ tfsdk.Resource = rssFeedResource{}
var _ tfsdk.ResourceWithImportState = rssFeedResource{}

type rssFeedResourceType struct{}

func (t rssFeedResourceType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: "Manage your rss feeds",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:        types.StringType,
				Description: "RSS feed id.",
				Computed:    true,
			},
			"title": {
				Type:        types.StringType,
				Description: "Title of the RSS feed as will appear on the site.",
				Required:    true,
			},
			"rss_source_url": {
				Type:        types.StringType,
				Description: "The URL of the RSS feed to be watched.",
				Required:    true,
			},
			"parent_dir_id": {
				Type:        types.Int64Type,
				Description: "The file ID of the folder to place the RSS feed files in.",
				Optional:    true,
				Computed:    true, // mandatory for default value
				PlanModifiers: []tfsdk.AttributePlanModifier{
					tfsdk.UseStateForUnknown(),
					modifiers.In64Default(0),
				},
			},
			"delete_old_files": {
				Type:        types.BoolType,
				Description: "Should old files in the folder be deleted when space is low.",
				Optional:    true,
				Computed:    true, // mandatory for default value
				PlanModifiers: []tfsdk.AttributePlanModifier{
					tfsdk.UseStateForUnknown(),
					modifiers.BoolDefault(false),
				},
			},
			// "dont_process_whole_feed": {
			//	Type:        types.BoolType,
			//	Description: "Should the current items in the feed, at creation time, be ignored.",
			//	Optional:    true,
			//	Computed:    true, // mandatory for default value
			//	PlanModifiers: []tfsdk.AttributePlanModifier{
			//		tfsdk.UseStateForUnknown(),
			//		modifiers.BoolDefault(false),
			//	},
			//},
			"keyword": {
				Type:        types.StringType,
				Description: "Only items with titles that contain any of these words will be transferred (comma-separated list of words).",
				Required:    true,
			},
			"unwanted_keywords": {
				Type:        types.StringType,
				Description: "No items with titles that contain any of these words will be transferred (comma-separated list of words).",
				Optional:    true,
				Computed:    true, // mandatory for default value
				PlanModifiers: []tfsdk.AttributePlanModifier{
					modifiers.StringDefault(""),
				},
			},
			"paused": {
				Type:        types.BoolType,
				Description: "Should the RSS feed be created in the paused state.",
				Optional:    true,
				Computed:    true, // mandatory for default value
				PlanModifiers: []tfsdk.AttributePlanModifier{
					modifiers.BoolDefault(false),
				},
			},
		},
	}, nil
}

func (t rssFeedResourceType) NewResource(_ context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return rssFeedResource{
		provider: provider,
	}, diags
}

// rssFeedResourceData describes an RSS feed as Terraform attributes.
type rssFeedResourceData struct {
	ID             types.String `json:"id,omitempty" tfsdk:"id"`
	Title          types.String `json:"title,omitempty" tfsdk:"title"`
	RssSourceURL   types.String `json:"rss_source_url,omitempty" tfsdk:"rss_source_url"`
	ParentDirID    types.Int64  `json:"parent_dir_id,omitempty" tfsdk:"parent_dir_id"`
	DeleteOldFiles types.Bool   `json:"delete_old_files,omitempty" tfsdk:"delete_old_files"`
	// DontProcessWholeFeed types.Bool   `json:"dont_process_whole_feed,omitempty" tfsdk:"dont_process_whole_feed"`
	Keyword          types.String `json:"keyword,omitempty" tfsdk:"keyword"`
	UnwantedKeywords types.String `json:"unwanted_keywords,omitempty" tfsdk:"unwanted_keywords"`
	Paused           types.Bool   `json:"paused,omitempty" tfsdk:"paused"`
}

// putioFeedData describes an RSS feed as Put.io REST format.
type putioFeedData struct {
	Feed struct {
		ID               int    `json:"id,omitempty"`
		Title            string `json:"title,omitempty"`
		RssSourceURL     string `json:"rss_source_url,omitempty"`
		ParentDirID      int    `json:"parent_dir_id,omitempty"`
		DeleteOldFiles   bool   `json:"delete_old_files,omitempty"`
		Keyword          string `json:"keyword,omitempty"`
		UnwantedKeywords string `json:"unwanted_keywords,omitempty"`
		Paused           bool   `json:"paused,omitempty"`
	} `json:"feed,omitempty"`
	Status string `json:"status,omitempty"`
}

type rssFeedResource struct {
	provider provider
}

func (r rssFeedResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data rssFeedResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// make feed to create
	payload := url.Values{}
	payload.Set("title", data.Title.Value)
	payload.Set("rss_source_url", data.RssSourceURL.Value)
	payload.Set("parent_dir_id", strconv.FormatInt(data.ParentDirID.Value, 10))
	payload.Set("delete_old_files", fmt.Sprintf("%t", data.DeleteOldFiles.Value))
	payload.Set("dont_process_whole_feed", "false") // TODO: support this
	payload.Set("keyword", data.Keyword.Value)
	payload.Set("unwanted_keywords", data.UnwantedKeywords.Value)
	payload.Set("paused", fmt.Sprintf("%t", data.Paused.Value))

	request, err := r.provider.client.NewRequest(ctx, http.MethodPost, "/v2/rss/create", strings.NewReader(payload.Encode()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create rss feed request, got error: %s", err))
		return
	}

	var createdFeed putioFeedData
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Content-Length", strconv.Itoa(len(payload.Encode())))
	if _, err := r.provider.client.Do(request, &createdFeed); err != nil {
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create rss feed, got error: %s", err))
			return
		}
	}

	data.ID = types.String{Value: strconv.Itoa(createdFeed.Feed.ID)}
	data.Title = types.String{Value: createdFeed.Feed.Title}
	data.RssSourceURL = types.String{Value: createdFeed.Feed.RssSourceURL}
	data.ParentDirID = types.Int64{Value: int64(createdFeed.Feed.ParentDirID)}
	data.DeleteOldFiles = types.Bool{Value: createdFeed.Feed.DeleteOldFiles}
	data.Keyword = types.String{Value: createdFeed.Feed.Keyword}
	data.UnwantedKeywords = types.String{Value: createdFeed.Feed.UnwantedKeywords}
	data.Paused = types.Bool{Value: createdFeed.Feed.Paused}

	// write logs using the tflog package
	// see https://pkg.go.dev/github.com/hashicorp/terraform-plugin-log/tflog
	// for more information
	tflog.Trace(ctx, "created a resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r rssFeedResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data rssFeedResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	request, err := r.provider.client.NewRequest(ctx, http.MethodGet, fmt.Sprintf("/v2/rss/%s", data.ID.Value), http.NoBody)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create rss feed request, got error: %s", err))
		return
	}

	var feedData putioFeedData
	if _, err := r.provider.client.Do(request, &feedData); err != nil {
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read rss feed, got error: %s", err))
			return
		}
	}

	data.ID = types.String{Value: strconv.Itoa(feedData.Feed.ID)}
	data.Title = types.String{Value: feedData.Feed.Title}
	data.RssSourceURL = types.String{Value: feedData.Feed.RssSourceURL}
	data.ParentDirID = types.Int64{Value: int64(feedData.Feed.ParentDirID)}
	data.DeleteOldFiles = types.Bool{Value: feedData.Feed.DeleteOldFiles}
	data.Keyword = types.String{Value: feedData.Feed.Keyword}
	data.UnwantedKeywords = types.String{Value: feedData.Feed.UnwantedKeywords}
	data.Paused = types.Bool{Value: feedData.Feed.Paused}

	// write logs using the tflog package
	// see https://pkg.go.dev/github.com/hashicorp/terraform-plugin-log/tflog
	// for more information
	tflog.Trace(ctx, "read a resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r rssFeedResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var data rssFeedResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// example, err := d.provider.client.UpdateExample(...)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r rssFeedResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data rssFeedResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	request, err := r.provider.client.NewRequest(ctx, http.MethodPost, fmt.Sprintf("/v2/rss/%s/delete", data.ID.Value), http.NoBody)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create rss feed request deletiob, got error: %s", err))
		return
	}

	var feedData putioFeedData
	if _, err := r.provider.client.Do(request, &feedData); err != nil {
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete rss feed, got error: %s", err))
			return
		}
	}
}

func (r rssFeedResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}
