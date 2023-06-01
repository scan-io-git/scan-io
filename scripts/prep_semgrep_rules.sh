#! /usr/bin/env bash

set -o errexit   # abort on nonzero exitstatus
set -o nounset   # abort on unbound variable
set -o pipefail  # don't hide errors within pipes

Help() {
   # Display Help
   echo "The script prepares semgrep rules, by fetching remote repositories and copying rules into `/rules` folder."
   echo "Be sure before using the script that you deleted a ../rules folder."
   echo
   echo "Syntax: prep_semgrep_rules.sh [-h|-d|-r rules_semgrep.txt https://github.com/returntocorp/semgrep-rules.git release |-m /path1 /path2|a /rules_semgrep.txt /rules_trailofbits.txt]"
   echo "options:"
   echo "-h     Print this Help."
   echo "-d     Debug output."
   echo "-r     Path to a rules list. Only rules from the list will be fetched. For example: -r example_rules.txt"
#    echo "-e     Path to a exclude list with rules. It creates a diff between your deleted rules and existed rule and new rules in the repo."
# ./prep_semgrep_rules.sh -e rules_semgrep.txt deleted_semgrep.txt https://github.com/returntocorp/semgrep-rules.git release
   echo "-m     Mergin two directories."
   echo "-a     Full auto mod. Downloads semgrep rules repo and trailofbits repo by lists. Mergin two dirictories."
 
   echo
}

REPOSITORY="https://github.com/returntocorp/semgrep-rules.git";
BRANCH="release";
RULES_LIST="rules_semgrep.txt";
MODE="rules";

parse_url() {
        local ARGS=("$@")
        local REPOSITORY_TO_PARS="${ARGS[0]}"
        # extract the protocol
        proto="$(echo $REPOSITORY_TO_PARS | grep :// | sed -e's,^\(.*://\).*,\1,g')"
        # remove the protocol
        url="$(echo ${REPOSITORY_TO_PARS/$proto/})"
        # extract the path
        path="$(echo $url | grep / | cut -d/ -f2-)"
        BASE_FOLDER="$(echo $path | awk -F / '{ print $(NR) }')"  
}

parse_url $REPOSITORY

### Args handler
while getopts hdmar:e: flag
do
   case "${flag}" in
     h) # display Help
         Help
         exit;;
     d) 
         echo "Debug mode"
         set -x;;  # debug output feature
     r)
         RULES_LIST=${OPTARG}
         REPOSITORY=$3
         BRANCH=$4
         MODE="rules"

         parse_url $REPOSITORY
        
         echo "Rule list: $RULES_LIST"
         echo "Repository with rules: $REPOSITORY"
         echo "Branch: $BRANCH";;
    #  e)
    #     RULES_LIST=${OPTARG}
    #     EXCLUDE_LIST=$3
    #     REPOSITORY=$4
    #     BRANCH=$5
    #     MODE="diff"

    #     parse_url $REPOSITORY
        
    #     echo "Rule list: $RULES_LIST"
    #     echo "Exclude list: $EXCLUDE_LIST"
    #     echo "Repository with rules: $REPOSITORY"
    #     echo "Branch: $BRANCH";;
     m) 
         MERGE_PATH=$2
         MODE="merge"
         echo "Merging rules $MERGE_PATH";;
     a)
         MODE="auto"
         if [ $# -ge 2 ] && [ -n "$2" ]
            then
            SEMGREP_FILE_WITH_RULES=$2
            echo "Semgrep files with rules is found - $SEMGREP_FILE_WITH_RULES"
        else
            SEMGREP_FILE_WITH_RULES="rules_semgrep.txt"
        fi
        if [ $# -ge 3 ] && [ -n "$3" ]
            then
            TRAILOFBITS_FILE_WITH_RULES=$3
            echo "Trailofbits files with rules is found - $TRAILOFBITS_FILE_WITH_RULES"
        else
            TRAILOFBITS_FILE_WITH_RULES="rules_trailofbits.txt"
        fi
         echo "Full auto mod. For a rules source will be used $SEMGREP_FILE_WITH_RULES and $TRAILOFBITS_FILE_WITH_RULES";;
     \?) # incorrect option
         echo "Error: Invalid option"
         exit;;
   esac
done


handle_rules_from_file() {
    local ARGS=("$@")
    local RULES_FILE="${ARGS[0]}"
    rules=()
    while IFS= read -r line || [[ -n "$line" ]]; do
        echo "Text read from a rules file: $line"
        rules+=("$line")
    done < $RULES_FILE
}

handle_repo() {
    local ARGS=("$@")
    local REPO="${ARGS[0]}"
    local BRANCH_OR_HASH="${ARGS[1]}"
    local RULES_FILE_FOR_CP="${ARGS[2]}"
    local RULES="${ARGS[@]:3}"

    local REPO_DIR=$(mktemp -d)
    echo "Directory of repository - $REPO_DIR"
    git clone $REPO $REPO_DIR

    pushd "${REPO_DIR}"
    # git pull origin $BRANCH_OR_HASH
    git checkout origin/$BRANCH_OR_HASH
    popd

    for rule in ${RULES[@]}; do
        local TARGET_FILE="$RULES_DIR/$rule"
        local TARGET_FOLDER=$( dirname -- $TARGET_FILE )
        mkdir -p $TARGET_FOLDER
        cp "$REPO_DIR/$rule" $TARGET_FILE
    done
    cp "$RULES_FILE_FOR_CP" "$RULES_DIR"
    
    if [ "$MODE" = "diff" ];
    then
        handle_rules_from_file $EXCLUDE_LIST
        echo "Exclude list: ${rules[@]}"
        handle_diff $RULES_DIR $REPO_DIR
        
        
    fi

    echo "Deleting tempo directory with cloned rules $REPO_DIR "
    rm -rf $REPO_DIR 
}

init_for_rules() {
    SCRIPT_DIR=$( dirname -- "$( readlink -f -- "$0"; )"; )
    PARENT_DIR=$( dirname -- $SCRIPT_DIR )
    RULES_DIR="$PARENT_DIR/rules/$BASE_FOLDER"
    echo "Rules directory to safe files - $RULES_DIR"
}

handle_merging() {
    local ARGS=("$@")
    local MERGE_PATH="${ARGS[0]}"
    MERGE_FOLDER="${MERGE_PATH%/*}/semgrep-rules"
    echo "Merging $MERGE_PATH to $MERGE_FOLDER"
    mkdir -p $MERGE_FOLDER
    cp -R $MERGE_PATH/ $MERGE_FOLDER 
    echo "Deleting $MERGE_PATH"
    rm -rf $MERGE_PATH
}

# handle_diff() {
#     local ARGS=("$@")
#     local FOLDER_WITH_LISTED_RULES="${ARGS[0]}"
#     local FOLDER_WITH_FETCHED_REPO="${ARGS[1]}"
#     echo $FOLDER_WITH_LISTED_RULES
#     echo $FOLDER_WITH_FETCHED_REPO
#     local CCC=$(diff -Nqr $FOLDER_WITH_LISTED_RULES $FOLDER_WITH_FETCHED_REPO | grep ".yaml")

#     for el in ${CCC[@]}; do
#         if [[ "$el" == *".yaml" ]] || [[ "$el" == *".yml" ]]
#         then  
#             PARSED_RULES=$(echo "$el" | awk -F / '{print $(NF-1)"/"$NF}')   
#             if [[ " ${rules[@]} " =~ "$PARSED_RULES" ]]
#                 then 
#                     echo "$el Not in Array"
#             else
#                 echo "$el is a New rule"
#             fi
#         fi
#     done
# }

case $MODE in
        "rules")
             init_for_rules
             handle_rules_from_file $RULES_LIST
             handle_repo $REPOSITORY $BRANCH $RULES_LIST ${rules[@]};;
        # "diff")
        #      init_for_rules
        #      handle_rules_from_file $RULES_LIST
        #      handle_repo $REPOSITORY $BRANCH $RULES_LIST ${rules[@]};;
        "merge")
             handle_merging $MERGE_PATH;;
        "auto")
             parse_url $REPOSITORY
             init_for_rules
             handle_rules_from_file $SEMGREP_FILE_WITH_RULES
             handle_repo $REPOSITORY $BRANCH $SEMGREP_FILE_WITH_RULES ${rules[@]}
             MERGE_PATH=$RULES_DIR
             handle_merging $MERGE_PATH

             REPOSITORY="https://github.com/trailofbits/semgrep-rules.git"
             BRANCH="main"

             parse_url $REPOSITORY
             init_for_rules
             handle_rules_from_file $TRAILOFBITS_FILE_WITH_RULES
             handle_repo $REPOSITORY $BRANCH $TRAILOFBITS_FILE_WITH_RULES ${rules[@]}
             
             MERGE_PATH=$RULES_DIR
             handle_merging $MERGE_PATH;;
esac


