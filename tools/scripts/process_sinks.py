import os
from pathlib import Path
from ruamel.yaml import YAML

def sort_field_properties(field):
    if not isinstance(field, dict):
        return field
    
    # Define the order for field properties
    field_order = ['path', 'name', 'label', 'kind', 'type', 'default', 'optional', 'examples', 'description']
    
    # Create a sorted dictionary with known properties first
    sorted_field = {}
    for key in field_order:
        if key in field:
            sorted_field[key] = field[key]
    
    # Add any remaining properties
    for key in field:
        if key not in field_order:
            sorted_field[key] = field[key]
    
    # If there are nested fields, sort them too
    if 'fields' in sorted_field:
        sorted_field['fields'] = [sort_field_properties(f) for f in sorted_field['fields']]
    
    return sorted_field

def sort_definition(content):
    if not isinstance(content, dict):
        return content
    
    # Define the order for root level properties
    root_order = ['model_version', 'kind', 'label', 'name', 'icon', 'status', 'description', 'fields']
    
    # Create a sorted dictionary with known properties first
    sorted_content = {}
    for key in root_order:
        if key in content:
            sorted_content[key] = content[key]
    
    # Add any remaining properties
    for key in content:
        if key not in root_order:
            sorted_content[key] = content[key]
    
    # Sort fields if they exist
    if 'fields' in sorted_content:
        sorted_content['fields'] = [sort_field_properties(f) for f in sorted_content['fields']]
    
    return sorted_content

def process_yaml_files(source_dir, target_dir):
    if not os.path.exists(source_dir):
        print(f"Source directory {source_dir} does not exist, skipping...")
        return

    # Create target directory if it doesn't exist
    Path(target_dir).mkdir(parents=True, exist_ok=True)
    
    # Process each YAML file
    for filename in os.listdir(source_dir):
        if filename.endswith(('.yml', '.yaml')):
            source_path = os.path.join(source_dir, filename)
            target_path = os.path.join(target_dir, filename)
            
            # Read and parse source file
            with open(source_path, 'r') as f:
                yaml = YAML()
                content = yaml.load(f)
            
            # Sort the content
            sorted_content = sort_definition(content)
            
            # Write to target file
            with open(target_path, 'w') as f:
                yaml = YAML()
                yaml.indent(mapping=2, sequence=4, offset=2)
                yaml.dump(sorted_content, f)
            print(f"Processed {filename}")

if __name__ == '__main__':
    # Get the project root directory (two levels up from the script)
    base_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
    connect_dir = os.path.join(base_dir, '.connect')
    
    # List of component types to process
    components = ['sources', 'sinks', 'scanners']
    
    # Process each component directory
    for component in components:
        source_dir = os.path.join(connect_dir, component)
        target_dir = os.path.join(connect_dir, component)
        print(f"\nProcessing {component} directory...")
        process_yaml_files(source_dir, target_dir)
