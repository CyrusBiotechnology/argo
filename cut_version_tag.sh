#!/usr/bin/env bash


while [[ $# -gt 0 ]]; do
    case "$1" in
        --environment=*)
          environment="${1#*=}"
          ;;
        *)

          echo "Error: --environment must be specified"
          exit 1
    esac
    shift
done

if ! [[ -n "$environment" ]]
then
    echo "Error: --environment must be specified"
    exit 1
fi

echo "checking out master branch"
git checkout master
echo "pulling latest changes"
git pull



version=`cat VERSION`
full_version="$version-$environment"

echo "current git HEAD is \"$(git log --oneline |head -1)\""
read -p "Would you like to create and push the tag ${full_version} at the current head of the master branch? (y/n)" proceed

if [[ ${proceed} == "y" ]]; then
    git tag ${full_version}
    git push --tags
fi