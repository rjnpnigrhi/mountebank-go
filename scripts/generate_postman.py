import os
import json
import glob
import re

TEMPLATE_DIR = "mountebank-templates"
OUTPUT_FILE = "mountebank_collection.json"

def parse_imposter(filepath):
    with open(filepath, 'r') as f:
        content = f.read()
    
    # Simple comment stripping (// ...)
    content = re.sub(r'//.*', '', content)
    # Remove leading/trailing whitespace
    content = content.strip()

    try:
        data = json.loads(content)
        return data
    except json.JSONDecodeError as e:
        print(f"Skipping {filepath}: Not valid JSON ({e})")
        return None

def extract_path(predicate):
    # Iterate through keys like 'equals', 'matches', 'startsWith'
    for operator in ['equals', 'deepEquals', 'contains', 'startsWith', 'endsWith', 'matches']:
        if operator in predicate:
            obj = predicate[operator]
            if 'path' in obj:
                return obj['path']
    return None

def extract_method(predicate):
    for operator in ['equals', 'matches', 'startsWith']:
        if operator in predicate:
            obj = predicate[operator]
            if 'method' in obj:
                return obj['method']
    return "GET" # Default

def main():
    collection = {
        "info": {
            "name": "Mountebank Stubs",
            "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
        },
        "item": []
    }

    files = glob.glob(os.path.join(TEMPLATE_DIR, "*.ejs"))
    
    for filepath in files:
        filename = os.path.basename(filepath)
        if filename == "imposters.ejs":
            continue

        print(f"Processing {filename}...")
        data = parse_imposter(filepath)
        if not data:
            continue

        port = data.get('port')
        name = data.get('name', f"Imposter {port}")
        protocol = data.get('protocol', 'http')
        
        if protocol != 'http' and protocol != 'https':
            print(f"Skipping {filename}: Protocol {protocol} not supported for Postman generation yet")
            continue

        host = "localhost"
        base_url = f"{protocol}://{host}:{port}"

        imposter_folder = {
            "name": name,
            "item": []
        }

        stubs = data.get('stubs', [])
        for i, stub in enumerate(stubs):
            predicates = stub.get('predicates', [])
            path = None
            method = "GET"

            for pred in predicates:
                p = extract_path(pred)
                if p:
                    path = p
                
                m = extract_method(pred)
                if m != "GET": # Update if found specific
                    method = m

            if not path:
                path = "/" 
                name = f"Stub {i+1} (Root)"
            else:
                name = f"{method} {path}"

            request_item = {
                "name": name,
                "request": {
                    "method": method,
                    "url": {
                        "raw": f"{base_url}{path}",
                        "protocol": protocol,
                        "host": [host],
                        "port": str(port),
                        "path": path.strip('/').split('/')
                    }
                }
            }
            imposter_folder["item"].append(request_item)

        if imposter_folder["item"]:
            collection["item"].append(imposter_folder)

    with open(OUTPUT_FILE, 'w') as f:
        json.dump(collection, f, indent=4)
    
    print(f"Successfully generated {OUTPUT_FILE}")

if __name__ == "__main__":
    main()
