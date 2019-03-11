#!/bin/bash

set -x

LSB_FILE="/etc/lsb-release.host"
OS_RELEASE_FILE="/etc/os-release.host"
TARGET_DIR="/usr/src"
HOST_MODULES_DIR="/lib/modules.host"

generate_headers()
{
  echo "Generating kernel headers"
  cd ${BUILD_DIR}
  zcat /proc/config.gz > .config
  make ARCH=x86 oldconfig > /dev/null
  make ARCH=x86 prepare > /dev/null

  # Clean up abundant non-header files to speed-up copying
  find ${BUILD_DIR} -regex '.*\.c\|.*\.txt\|.*Makefile\|.*Build\|.*Kconfig' -type f -delete
}

fetch_cos_linux_sources()
{
  echo "Fetching upstream kernel sources."
  mkdir -p ${BUILD_DIR}
  curl -s "https://storage.googleapis.com/cos-tools/${BUILD_ID}/kernel-src.tar.gz" | tar -xzf - -C ${BUILD_DIR}
}

fetch_generic_linux_sources()
{
  kernel_version=$(uname -r | tr -d '+')
  major_version=$(echo ${kernel_version} | cut -d . -f 1)
  echo "Fetching upstream kernel sources for ${kernel_version}."
  mkdir -p ${BUILD_DIR}
  curl -sL https://www.kernel.org/pub/linux/kernel/v${major_version}.x/linux-$kernel_version.tar.gz | tar --strip-components=1 -xzf - -C ${BUILD_DIR}

}

install_cos_linux_headers()
{
  if grep -q CHROMEOS_RELEASE_VERSION ${LSB_FILE};then
    BUILD_ID=$(grep CHROMEOS_RELEASE_VERSION ${LSB_FILE} | cut -d = -f 2)
    BUILD_DIR="/linux-lakitu-${BUILD_ID}"
    SOURCES_DIR="${TARGET_DIR}/linux-lakitu-${BUILD_ID}"

    if [ ! -e "${SOURCES_DIR}/.installed" ];then
      echo "Installing kernel headers for COS build ${BUILD_ID}"
      time fetch_cos_linux_sources
      time generate_headers
      time mv ${BUILD_DIR} ${TARGET_DIR}
      touch "${SOURCES_DIR}/.installed"
    fi
  fi
}

install_generic_linux_headers()
{
  BUILD_DIR="/linux-generic-$(uname -r)"
  SOURCES_DIR="${TARGET_DIR}/linux-generic-$(uname -r)"

  if [ ! -e "${SOURCES_DIR}/.installed" ];then
    echo "Installing kernel headers for generic kernel"
    time fetch_generic_linux_sources
    time generate_headers
    time mv ${BUILD_DIR} ${TARGET_DIR}
    touch "${SOURCES_DIR}/.installed"
  fi
}

install_headers()
{
  distro=$(grep ^NAME ${OS_RELEASE_FILE} | cut -d = -f 2)

  case $distro in
    *"Container-Optimized OS"*)
      install_cos_linux_headers
      HEADERS_TARGET=${SOURCES_DIR}
      ;;
    *)
      echo "WARNING: Cannot find distro-specific headers for ${distro}. Fetching generic headers."
      install_generic_linux_headers
      HEADERS_TARGET=${SOURCES_DIR}
      ;;
  esac
}

check_headers()
{
  modules_path=$1
  utsname=$(uname -r)
  arch=$(uname -m)
  kdir="${modules_path}/${utsname}"

  [ "${arch}" == "x86_64" ] && arch="x86"

  [ ! -e ${kdir} ] && return 1
  [ ! -e "${kdir}/source" ] && [ ! -e "${kdir}/build" ] && return 1

  header_dir=$([ -e "${kdir}/source" ] && echo "${kdir}/source" || echo "${kdir}/build")

  [ ! -e "${header_dir}/include/linux/kconfig.h" ] && return 1
  [ ! -e "${header_dir}/include/generated/uapi" ] && return 1
  [ ! -e "${header_dir}/arch/${arch}/include/generated/uapi" ] && return 1

  return 0
}

if [ ! -e /lib/modules/.installed ];then
  if ! check_headers ${HOST_MODULES_DIR}; then
    install_headers
  else
    HEADERS_TARGET=${HOST_MODULES_DIR}/source
  fi

  mkdir -p "/lib/modules/$(uname -r)"
  ln -sf ${HEADERS_TARGET} "/lib/modules/$(uname -r)/source"
  ln -sf ${HEADERS_TARGET} "/lib/modules/$(uname -r)/build"
  touch /lib/modules/.installed
  exit 0
else
  echo "Headers already installed"
  exit 0
fi
