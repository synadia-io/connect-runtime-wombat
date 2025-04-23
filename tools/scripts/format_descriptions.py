import os
from ruamel.yaml import YAML
from ruamel.yaml.scalarstring import LiteralScalarString

def format_descriptions(data):
    """Recursively format all description fields in the YAML data."""
    if isinstance(data, dict):
        for key, value in data.items():
            if key == 'description' and isinstance(value, str):
                # Remove any existing block indicators and extra whitespace
                cleaned = value.replace('\\n', '\n').strip()
                if cleaned.startswith('|'):
                    cleaned = cleaned[1:].strip()
                # Set as a literal block
                data[key] = LiteralScalarString(cleaned)
            elif isinstance(value, (dict, list)):
                format_descriptions(value)
    elif isinstance(data, list):
        for item in data:
            if isinstance(item, (dict, list)):
                format_descriptions(item)

def update_yaml_files(directory):
    yaml = YAML()
    yaml.preserve_quotes = True
    yaml.indent(mapping=2, sequence=4, offset=2)
    
    if not os.path.exists(directory):
        print(f"Directory {directory} does not exist, skipping...")
        return
        
    for filename in os.listdir(directory):
        if filename.endswith(('.yml', '.yaml')):
            filepath = os.path.join(directory, filename)
            with open(filepath, 'r') as f:
                data = yaml.load(f)
            
            # Ensure model_version is the first key
            if not isinstance(data, dict) or 'model_version' not in data:
                new_data = {'model_version': '1'}
                if isinstance(data, dict):
                    new_data.update(data)
                data = new_data
            
            # Format all descriptions
            format_descriptions(data)
            
            # Write back to file, preserving the formatting
            with open(filepath, 'w') as f:
                yaml.dump(data, f)
            print(f"Updated {filename}")

if __name__ == '__main__':
    # Get the project root directory (two levels up from the script)
    base_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
    connect_dir = os.path.join(base_dir, '.connect')
    
    # List of component types to process
    components = ['sources', 'sinks', 'scanners']
    
    # Process each component directory
    for component in components:
        component_dir = os.path.join(connect_dir, component)
        print(f"\nProcessing {component} directory...")
        update_yaml_files(component_dir)
