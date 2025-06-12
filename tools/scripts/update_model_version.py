import os
import yaml

def update_yaml_files(directory):
    if not os.path.exists(directory):
        print(f"Directory {directory} does not exist, skipping...")
        return

    for filename in os.listdir(directory):
        if filename.endswith(('.yml', '.yaml')):
            filepath = os.path.join(directory, filename)
            with open(filepath, 'r') as f:
                content = f.read()

            # Parse YAML content
            data = yaml.safe_load(content)

            # Add model_version if it's not already the first key
            if not isinstance(data, dict) or list(data.keys())[0] != 'model_version':
                # Create new dict with model_version first
                new_data = {'model_version': '1'}
                if isinstance(data, dict):
                    new_data.update(data)

                # Write back to file
                with open(filepath, 'w') as f:
                    yaml.dump(new_data, f, default_flow_style=False, sort_keys=False)
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
