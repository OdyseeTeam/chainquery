# \DefaultApi

All URIs are relative to *http://localhost/api*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddressSummary**](DefaultApi.md#AddressSummary) | **Get** /AddressSummary | Returns a summary of Address activity
[**Status**](DefaultApi.md#Status) | **Get** /Status | Returns important status information about Chain Query


# **AddressSummary**
> AddressSummary AddressSummary($address)

Returns a summary of Address activity

It returns sent, recieved, balance, and number of transactions it has been used in.


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **address** | **string**| LBRY Address you would like a summary on. | 

### Return type

[**AddressSummary**](AddressSummary.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **Status**
> TableStatus Status()

Returns important status information about Chain Query


### Parameters
This endpoint does not need any parameter.

### Return type

[**TableStatus**](TableStatus.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

