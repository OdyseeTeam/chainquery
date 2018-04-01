# swagger_client.DefaultApi

All URIs are relative to *http://localhost/api*

Method | HTTP request | Description
------------- | ------------- | -------------
[**address_summary**](DefaultApi.md#address_summary) | **GET** /AddressSummary | Returns a summary of Address activity
[**status**](DefaultApi.md#status) | **GET** /Status | Returns important status information about Chain Query


# **address_summary**
> AddressSummary address_summary(address)

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
api_instance = swagger_client.DefaultApi()
address = 'address_example' # str | LBRY Address you would like a summary on.

try: 
    # Returns a summary of Address activity
    api_response = api_instance.address_summary(address)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling DefaultApi->address_summary: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **address** | **str**| LBRY Address you would like a summary on. | 

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
api_instance = swagger_client.DefaultApi()

try: 
    # Returns important status information about Chain Query
    api_response = api_instance.status()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling DefaultApi->status: %s\n" % e)
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

