# swagger_client.QueryApi

All URIs are relative to *http://:6300/api*

Method | HTTP request | Description
------------- | ------------- | -------------
[**s_ql_query**](QueryApi.md#s_ql_query) | **GET** /SQL | Use SQL in a RESTful way


# **s_ql_query**
> list[object] s_ql_query(query)

Use SQL in a RESTful way

API exposed for sending SQL queries directly against the chainquery database. Since this is an exposed API there are limits to its use. These limits include queries per hour, read-only, limited to 60 second timeout. 

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.QueryApi()
query = 'query_example' # str | The SQL query to be put against the chainquery database.

try:
    # Use SQL in a RESTful way
    api_response = api_instance.s_ql_query(query)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling QueryApi->s_ql_query: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **query** | **str**| The SQL query to be put against the chainquery database. | 

### Return type

**list[object]**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

