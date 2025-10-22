#!/bin/bash
#set -xe
REPO_HOME=$(cd $(dirname $0) && pwd)
cd ${REPO_HOME}
OS_SYSTEM=$(uname)
if [ "$OS_SYSTEM" = "Darwin" ];
then
    brew_prefix=$(brew --prefix)
    python_installation_dir="${brew_prefix}/bin"
else
    python_installation_dir="/usr/bin"
fi


function is_python_version_suported () {
    supported_python_major_minor_versions=("3.9 3.10")
    input_version="${1}"

    if [[ " ${supported_python_major_minor_versions[*]} " =~ " ${input_version} " ]]; then
        echo "python input_version : ${input_version} is supported  "
        return 0
    elif [[ ! " ${supported_python_major_minor_versions[*]} " =~ " ${input_version} " ]]; then
        echo "python input_version : ${input_version} is not supported, supported versions are ${supported_python_major_minor_versions} "
        return 1
    fi
}

function is_python_valid () {
    python3_path="${1}"
    if [ -f "${python3_path}" ]; then
        echo "${python3_path} exists."
        if [ -L "${python3_path}" ] ; then
            if [ -e "${python3_path}" ] ; then
                echo "${python3_path} is a Good symlink"
                python3_full_version=$(${python3_path} -V | awk '{ print $2 }' )
                python3_major_version=$(${python3_path} -V | awk '{ print $2 }' | awk -F. '{ print $1 }')
                python3_minor_version=$(${python3_path} -V | awk '{ print $2 }' | awk -F. '{ print $2 }')
                python3_patch_version=$(${python3_path} -V | awk '{ print $2 }' | awk -F. '{ print $3 }')
                python3_major_minor_version=$(echo "${python3_major_version}.${python3_minor_version}" | xargs )
                if is_python_version_suported "${python3_major_minor_version}" ; then
                    return 0
                else
                    return 1
                fi
            else
                echo "${python3_path} is a Broken symlink"
            fi
        elif [ -e ${python3_path} ] ; then
            echo "${python3_path} is Not a symlink"
            python3_full_version=$(${python3_path} -V | awk '{ print $2 }' )
            python3_major_version=$(${python3_path} -V | awk '{ print $2 }' | awk -F. '{ print $1 }')
            python3_minor_version=$(${python3_path} -V | awk '{ print $2 }' | awk -F. '{ print $2 }')
            python3_patch_version=$(${python3_path} -V | awk '{ print $2 }' | awk -F. '{ print $3 }')
            python3_major_minor_version=$(echo "${python3_major_version}.${python3_minor_version}" | xargs )
            if is_python_version_suported "${python3_major_minor_version}" ; then
                return 0
            else
                return 1
            fi
        else
            echo "${python3_path} is Missing"
            return 1
        fi
    else
        echo "${python3_path} doesn't exist"
        return 1
    fi
}

default_python3_path="${python_installation_dir}/python3"
python3_command=$(find ${python_installation_dir} -name "python3.*" | grep -E "3.9$|3.10$" | head -n1)

if is_python_valid "${default_python3_path}" ; then
    python3_command=${default_python3_path}
fi

if [ -z "$python3_command" ]
then
    echo "Unable to find suitable python3 , python3.9 or higher is required"
    exit 1
elif is_python_valid "${python3_command}";
then
      echo "found ${python3_command} , will be using it to create the virtual env"
else
    echo "Unable to find suitable python3 , python3.9 or higher is required"
    exit 1
fi

if [[ ! -d "${REPO_HOME}/.venv" ]] ; then ${python3_command} -m venv "${REPO_HOME}/.venv"; fi
source "${REPO_HOME}/.venv/bin/activate"
which python
${REPO_HOME}/.venv/bin/pip install -r "${REPO_HOME}/requirements-test.txt"
#${REPO_HOME}/.venv/bin/pip install -r "${REPO_HOME}/requirements_dev.txt"
${REPO_HOME}/.venv/bin/pip install --upgrade pip
source ${REPO_HOME}/.venv/bin/activate
echo ""
echo ""
echo "please run : source ${REPO_HOME}/.venv/bin/activate"
echo ""
