# swagger_client.StatApi

All URIs are relative to *http://:6300/api*

Method | HTTP request | Description
------------- | ------------- | -------------
[**address_summary**](StatApi.md#address_summary) | **GET** /AddressSummary | Returns a summary of Address activity
[**status**](StatApi.md#status) | **GET** /Status | Returns important status information about Chain Query


# **address_summary**
> AddressSummary address_summary(lbry_address)

Returns a summary of Address activity

It returns sent, recieved, balance, and number of transactions it has been used in.

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.StatApi()
lbry_address = 'lbry_address_example' # str | A LbryAddress

try:
    # Returns a summary of Address activity
    api_response = api_instance.address_summary(lbry_address)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling StatApi->address_summary: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **lbry_address** | **str**| A LbryAddress | 

### Return type

[**AddressSummary**](AddressSummary.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **status**
> TableStatus status()

Returns important status information about Chain Query

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.StatApi()

try:
    # Returns important status information about Chain Query
    api_response = api_instance.status()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling StatApi->status: %s\n" % e)
```

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

