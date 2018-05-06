# \QueryApi

All URIs are relative to *http://:6300/api*

Method | HTTP request | Description
------------- | ------------- | -------------
[**SQLQuery**](QueryApi.md#SQLQuery) | **Get** /SQL | Use SQL in a RESTful way


# **SQLQuery**
> []interface{} SQLQuery(ctx, query)
Use SQL in a RESTful way

API exposed for sending SQL queries directly against the chainquery database. Since this is an exposed API there are limits to its use. These limits include queries per hour, read-only, limited to 60 second timeout. 

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **query** | **string**| The SQL query to be put against the chainquery database. | 

### Return type

[**[]interface{}**](interface{}.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

