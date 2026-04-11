import zipfile
import json

trace_path = "test-results/users-Users-live-should-create-a-new-user-chromium-retry1/trace.zip"
with zipfile.ZipFile(trace_path, 'r') as z:
    for name in z.namelist():
        if name.endswith('.network'):
            data = z.read(name)
            for line in data.decode('utf-8').split('\n'):
                if not line: continue
                try:
                    obj = json.loads(line)
                    if obj.get('method') == 'response' and 'users' in obj.get('params', {}).get('response', {}).get('url', ''):
                        if 'api/users' in obj['params']['response']['url']:
                          print("URL:", obj['params']['response']['url'])
                          print("Status:", obj['params']['response']['status'])
                except Exception as e:
                    pass
