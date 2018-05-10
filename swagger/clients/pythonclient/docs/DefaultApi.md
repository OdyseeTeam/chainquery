# swagger_client.DefaultApi

All URIs are relative to *http://0.0.0.0:6300/api*

Method | HTTP request | Description
------------- | ------------- | -------------
[**auto_update**](DefaultApi.md#auto_update) | **POST** /autoupdate | auto updates the application with the latest release based on TravisCI webhook


# **auto_update**
> auto_update(payload)

auto updates the application with the latest release based on TravisCI webhook

takes a webhook as defined by https://docs.travis-ci.com/user/notifications/#Webhooks-Delivery-Format, validates the public key, chooses whether or not update the application. If so it shuts down the api, downloads the latest release from https://github.com/lbryio/chainquery/releases, replaces the binary and starts the api again.

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
payload = NULL # object | 

try:
    # auto updates the application with the latest release based on TravisCI webhook
    api_instance.auto_update(payload)
except ApiException as e:
    print("Exception when calling DefaultApi->auto_update: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **payload** | **object**|  | 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/x-www-form-urlencoded
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

