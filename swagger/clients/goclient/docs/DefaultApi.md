# \DefaultApi

All URIs are relative to *http://0.0.0.0:6300/api*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AutoUpdate**](DefaultApi.md#AutoUpdate) | **Post** /autoupdate | auto updates the application with the latest release based on TravisCI webhook


# **AutoUpdate**
> AutoUpdate(ctx, payload)
auto updates the application with the latest release based on TravisCI webhook

takes a webhook as defined by https://docs.travis-ci.com/user/notifications/#Webhooks-Delivery-Format, validates the public key, chooses whether or not update the application. If so it shuts down the api, downloads the latest release from https://github.com/lbryio/chainquery/releases, replaces the binary and starts the api again.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **payload** | [**interface{}**](interface{}.md)|  | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/x-www-form-urlencoded
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

