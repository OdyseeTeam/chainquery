# \DefaultApi

All URIs are relative to *http://0.0.0.0:6300/api*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AutoUpdate**](DefaultApi.md#AutoUpdate) | **Get** /autoupdate | auto updates the application with the latest release based on TravisCI webhook


# **AutoUpdate**
> AutoUpdate(ctx, )
auto updates the application with the latest release based on TravisCI webhook

takes a webhook as defined by https://docs.travis-ci.com/user/notifications/#Webhooks-Delivery-Format, validates the public key, chooses whether or not update the application. If so it shuts down the api, downloads the latest release from https://github.com/lbryio/chainquery/releases, replaces the binary and starts the api again.

### Required Parameters
This endpoint does not need any parameter.

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

