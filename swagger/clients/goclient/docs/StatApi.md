# \StatApi

All URIs are relative to *http://0.0.0.0:6300/api*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddressSummary**](StatApi.md#AddressSummary) | **Get** /addresssummary | Returns a summary of Address activity
[**ChainQueryStatus**](StatApi.md#ChainQueryStatus) | **Get** /status | Returns important status information about Chain Query


# **AddressSummary**
> AddressSummary AddressSummary(ctx, lbryAddress)
Returns a summary of Address activity

It returns sent, recieved, balance, and number of transactions it has been used in.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for logging, tracing, authentication, etc.
  **lbryAddress** | **string**| A LbryAddress | 

### Return type

[**AddressSummary**](AddressSummary.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ChainQueryStatus**
> TableStatus ChainQueryStatus(ctx, )
Returns important status information about Chain Query

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**TableStatus**](TableStatus.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

