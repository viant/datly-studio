"""Exercise native authoring while the asynchronous Datly audit writer inserts."""
import concurrent.futures
import json
import pathlib
import sys
import urllib.error
import urllib.request

run_dir = pathlib.Path(sys.argv[1])
bearer = (run_dir / 'owner.token').read_text().strip()


def invoke(audit_call):
    if audit_call:
        url = 'http://127.0.0.1:19790/mcp'
        body = {'jsonrpc': '2.0', 'id': 1, 'method': 'tools/call',
                'params': {'name': 'studio.sdk.namespaces.list', 'arguments': {}}}
    else:
        url = 'http://127.0.0.1:19781/v1/studio/sdk/connectors.test'
        body = {'name': 'discovery_scoped'}
    headers = {'Authorization': 'Bearer ' + bearer,
               'Content-Type': 'application/json', 'Accept': 'application/json'}
    if audit_call:
        headers.update({'Mcp-Method': 'tools/call', 'Mcp-Name': 'studio.sdk.namespaces.list'})
    request = urllib.request.Request(url, data=json.dumps(body).encode(), headers=headers)
    try:
        with urllib.request.urlopen(request, timeout=15) as response:
            payload = json.load(response)
            if audit_call and (payload.get('error') or payload.get('result', {}).get('isError')):
                return 'tool-error'
            return response.status
    except urllib.error.HTTPError as error:
        return error.code


with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
    results = list(pool.map(invoke, [True, False] * 20))
failures = [result for result in results if result not in (200, 409)]
assert not failures, f'Concurrent audit/authoring failures: {failures}'
print('PASS: 40 concurrent MCP audit and authoring requests; no storage errors')
