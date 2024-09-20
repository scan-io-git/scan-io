import os
import shutil
import yaml
import git
import uuid
import tempfile
import argparse
import sys
from git import Repo
from colorama import Fore, Style, init
from tqdm import tqdm

def parse_args():
    """Parse command-line arguments."""
    script_dir = os.path.dirname(os.path.abspath(__file__))  
    default_rules_path = os.path.join(script_dir, 'scanio_rules.yaml') 
    
    parser = argparse.ArgumentParser(description="Build rule sets from a YAML configuration file.")
    parser.add_argument(
        "-r", "--rules", 
        type=str, 
        help=f"Path to the scanio_rules.yaml file. Defaults to '{default_rules_path}' in the script directory.",
        default=default_rules_path
    )
    parser.add_argument(
        "-f", "--force", 
        action="store_true", 
        help="Force clean the 'rules' directory without confirmation."
    )
    parser.add_argument(
        "--rules-dir", 
        type=str, 
        help="Path to the directory where rules will be stored. Defaults to './rules'.",
        default="rules"
    )
    parser.add_argument(
        "-v", "--verbose", 
        action="count", 
        default=0,
        help="Increase verbosity level. Use multiple times for more verbosity: -v, -vv."
    )
    parser.add_argument(
        "--no-color", 
        action="store_true", 
        help="Disable colored output."
    )
    return parser.parse_args()

def init_colorama(use_color=True):
    """Initialize colorama if color printing is enabled."""
    if use_color:
        init(autoreset=True)
    else:
        global Fore, Style
        Fore = Style = type('', (), {"RESET_ALL": '', "RED": '', "GREEN": '', "YELLOW": '', "CYAN": ''})()


def load_yaml(file_path):
    """Load and validate the YAML configuration file."""
    if not os.path.exists(file_path):
        raise FileNotFoundError(f"{Fore.RED}Error: {file_path} does not exist.{Style.RESET_ALL}")
    
    with open(file_path, 'r') as file:
        return yaml.safe_load(file)

def clean_rules_directory(rules_dir, force_clean=False):
    """Clean up the rules directory after user confirmation, with optional force cleaning."""
    rules_dir = os.path.abspath(rules_dir)

    if not os.path.exists(rules_dir):
        return  # If directory does not exist, no need to clean.

    files_in_dir = os.listdir(rules_dir)

    # Remove ignored files (.gitignore and .DS_Store)
    ignored_files = [".gitignore", ".DS_Store"]
    files_in_dir = [file for file in files_in_dir if file not in ignored_files]

    if not files_in_dir:
        print(f"{Fore.GREEN}Rules directory '{rules_dir}' contains only '.gitignore' and/or '.DS_Store', no cleanup needed.{Style.RESET_ALL}")
        return

    if force_clean:
        print(f"{Fore.YELLOW}Force cleaning the rules directory '{rules_dir}' (without confirmation).{Style.RESET_ALL}")
        for item in files_in_dir:
            item_path = os.path.join(rules_dir, item)
            if os.path.isdir(item_path):
                shutil.rmtree(item_path)
            else:
                os.remove(item_path)
        print(f"{Fore.GREEN}Cleaned up rules directory '{rules_dir}'.{Style.RESET_ALL}")
        return

    # Ask user for confirmation to delete remaining files
    print(f"{Fore.RED}rules directory '{rules_dir}' is not empty: {files_in_dir}{Style.RESET_ALL}")
    confirm = input(f"{Fore.YELLOW}Do you want to delete all files in '{rules_dir}' (except .gitignore and .DS_Store)? [y/N]: {Style.RESET_ALL}").strip().lower()

    if confirm == 'y':
        for item in files_in_dir:
            item_path = os.path.join(rules_dir, item)
            if os.path.isdir(item_path):
                shutil.rmtree(item_path)
            else:
                os.remove(item_path)
        print(f"{Fore.GREEN}Cleaned up rules directory '{rules_dir}'.{Style.RESET_ALL}")
    else:
        print(f"{Fore.CYAN}Proceeding without cleaning the rules directory '{rules_dir}'.{Style.RESET_ALL}")

def clone_repo(repo_url, branch, tmp_dir, tool, verbose):
    """Clone a repository to a temporary directory."""
    repo_tmp_path = os.path.join(tmp_dir, tool, str(uuid.uuid4()))
    if verbose >= 1:
        print(f"{Fore.CYAN}    Cloning {repo_url} (branch: {branch}) into {repo_tmp_path}{Style.RESET_ALL}")
    
    Repo.clone_from(repo_url, repo_tmp_path, branch=branch)
    return repo_tmp_path
    
def copy_files(paths, repo_tmp_path, ruleset_path, tool, ruleset, repo_url, branch, missing_files, overwritten_files, verbose):
    """Copy the specified files from the cloned repo to the ruleset directory."""
    for file_path in tqdm(paths, desc=f"{Fore.CYAN}      Processing{Style.RESET_ALL}", unit="file"):
        src_path = os.path.join(repo_tmp_path, file_path)
        dest_path = os.path.join(ruleset_path, file_path)

        # Ensure destination directory exists before copying
        os.makedirs(os.path.dirname(dest_path), exist_ok=True)

        if os.path.exists(src_path):
            if os.path.exists(dest_path):
                overwritten_files.append(f"Tool: {tool}, Ruleset: {ruleset}, File: {dest_path} (from {repo_url}, branch - {branch}) is overwritten")
                if verbose >= 2:
                    print(f"{Fore.YELLOW}      Overwriting {dest_path}{Style.RESET_ALL}")
            elif verbose >= 2:
                print(f"{Fore.GREEN}      Copied {file_path} to {dest_path}{Style.RESET_ALL}")
            shutil.copy(src_path, dest_path)
        else:
            missing_files.append(f"Tool: {tool}, Ruleset: {ruleset}, File: {file_path} not found in {repo_url}, branch - {branch}")
            if verbose >= 2:
                print(f"{Fore.RED}      Warning: {file_path} not found in {repo_url}{Style.RESET_ALL}")

def save_partial_yaml(tool, ruleset, repos, backup_file, verbose):
    """Save a portion of the YAML file for each ruleset."""
    partial_yaml = {'tools': {tool: {'rulesets': {ruleset: repos}}}}
    with open(backup_file, 'w') as f:
        yaml.dump(partial_yaml, f)
    
    if verbose >= 1:
        print(f"{Fore.GREEN}    Backup of tool-specific YAML saved to {backup_file}{Style.RESET_ALL}")

def save_full_yaml(original_yaml_path, rules_dir, verbose):
    """Save the full scanio_rules.yaml to the rules directory."""
    backup_file = os.path.join(rules_dir, "scanio_rules.yaml")
    shutil.copy(original_yaml_path, backup_file)
    
    if verbose >= 1:
        print(f"{Fore.GREEN}Backup of the entire scanio_rules.yaml saved to {backup_file}{Style.RESET_ALL}")

def print_warnings_and_overwrites(overwritten_files, missing_files, verbose):
    """Print lists of overwritten and missing files."""
    if overwritten_files:
        print(f"\n{Fore.YELLOW}Total overwritten files: {len(overwritten_files)}{Style.RESET_ALL}")
        if verbose >= 1:
            print(f"{Fore.YELLOW}The following files were overwritten during the bundling process:{Style.RESET_ALL}")
            for overwrite in overwritten_files:
                print(f"{Fore.YELLOW}  - {overwrite}{Style.RESET_ALL}")
    
    if missing_files:
        print(f"\n{Fore.RED}Total missing files: {len(missing_files)}{Style.RESET_ALL}")
        if verbose >= 1:
            print(f"{Fore.RED}The following files were not found during the bundling process:{Style.RESET_ALL}")
            for warning in missing_files:
                print(f"{Fore.RED}  - {warning}{Style.RESET_ALL}")

def process_rules(data, tmp_dir, rules_dir, original_yaml_path, verbose):
    """Process the tools and rulesets, clone repos, copy files, and log warnings."""
    overwritten_files = []
    missing_files = []
    errors = []

    for tool, tool_data in tqdm(data['tools'].items(), desc=f"{Fore.CYAN}Processing tools{Style.RESET_ALL}", unit="tool"):
        print(f"{Fore.CYAN}Processing tool: {tool}{Style.RESET_ALL}")
        tool_path = os.path.join(rules_dir, tool)
        os.makedirs(tool_path, exist_ok=True)

        for ruleset, repos in tool_data['rulesets'].items():
            print(f"{Fore.CYAN}  Processing ruleset: {ruleset}{Style.RESET_ALL}")
            ruleset_path = os.path.join(tool_path, ruleset)
            os.makedirs(ruleset_path, exist_ok=True)

            for repo_info in repos:
                repo_url = repo_info['repo']
                branch = repo_info['branch']
                paths = repo_info['paths']
                print(f"{Fore.CYAN}    Processing rules from: {repo_url}{Style.RESET_ALL}")

                try:
                    repo_tmp_path = clone_repo(repo_url, branch, tmp_dir, tool, verbose)
                    copy_files(paths, repo_tmp_path, ruleset_path, tool, ruleset, repo_url, branch, missing_files, overwritten_files, verbose)
                except git.GitCommandError as e:
                    print(f"{Fore.RED}Error cloning {repo_url}: {e}{Style.RESET_ALL}")
                    errors.append(f"Error cloning {repo_url}: {e}")
                    continue

            backup_file = os.path.join(ruleset_path, "scanio_rules.yaml.back")
            save_partial_yaml(tool, ruleset, repos, backup_file, verbose)
            print(f"    {Fore.GREEN}Finished processing ruleset: {ruleset}{Style.RESET_ALL}")
        print(f"{Fore.GREEN}Finished processing tool: {tool}{Style.RESET_ALL}")


    save_full_yaml(original_yaml_path, rules_dir, verbose)
    print_warnings_and_overwrites(overwritten_files, missing_files, verbose)

    return errors

if __name__ == "__main__":
    args = parse_args()

    # Initialize colorama based on the --no-color argument
    init_colorama(use_color=not args.no_color)

    try:
        # Load YAML configuration
        data = load_yaml(args.rules)

        # Clean up the rules directory if needed, with support for non-interactive mode
        clean_rules_directory(args.rules_dir, force_clean=args.force)

        # Create temporary directory for cloned repositories
        with tempfile.TemporaryDirectory() as tmp_dir:
            if args.verbose >= 1:
                print(f"{Fore.CYAN}Using temporary directory: {tmp_dir}{Style.RESET_ALL}")
            
            # Process the rules, clone repos, copy files
            handler_errors = process_rules(data, tmp_dir, args.rules_dir, args.rules, args.verbose)

        print(f"\n{Fore.GREEN}Temporary directory cleaned up automatically.{Style.RESET_ALL}")


        # Check if there were errors and print them, then exit with status 1 if errors exist
        if handler_errors:
            print(f"\n{Fore.RED}The following errors occurred during the bundling process:{Style.RESET_ALL}")
            for error in handler_errors:
                print(f"{Fore.RED}  - {error}{Style.RESET_ALL}")
            sys.exit(1)

    except FileNotFoundError as e:
        print(e)
        sys.exit(1)
    except Exception as e:
        print(f"{Fore.RED}Unexpected error: {e}{Style.RESET_ALL}")
        sys.exit(1)