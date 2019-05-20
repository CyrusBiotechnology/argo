#!/usr/bin/env bash

echo "checking out master branch"
git checkout master
echo "pulling latest changes"
git pull

version=`cat VERSION`
environment=$1
full_version="$version-$environment"

echo "current git HEAD is \"$(git log --oneline |head -1)\""
read -p "Would you like to create and push the tag ${full_version} at the current head of the master branch? (y/n)" proceed

if [[ ${proceed} == "y" ]]; then
    git tag ${full_version}
    git push --tags
fi