# swagger_client.UserApi

All URIs are relative to *http://api.lbry.io*

Method | HTTP request | Description
------------- | ------------- | -------------
[**user_new_get**](UserApi.md#user_new_get) | **GET** /user/new | Creates new user and returns an authtoken for interaction with the API


# **user_new_get**
> DefinitionsUser user_new_get()

Creates new user and returns an authtoken for interaction with the API



### Example 
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.UserApi()

try: 
    # Creates new user and returns an authtoken for interaction with the API
    api_response = api_instance.user_new_get()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling UserApi->user_new_get: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**DefinitionsUser**](DefinitionsUser.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

