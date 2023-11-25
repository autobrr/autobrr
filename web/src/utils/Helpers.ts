export const RandomLinuxIsos = (count: number) => {
  const linuxIsos = [
    "ubuntu-20.04.4-lts-focal-fossa-desktop-amd64-secure-boot",
    "debian-11.3.0-bullseye-amd64-DVD-1-with-nonfree-firmware-netinst",
    "fedora-36-workstation-x86_64-live-iso-with-rpmfusion-free-and-nonfree",
    "archlinux-2023.04.01-x86_64-advanced-installation-environment",
    "linuxmint-20.3-uma-cinnamon-64bit-full-multimedia-support-edition",
    "centos-stream-9-x86_64-dvd1-full-install-iso-with-extended-repositories",
    "opensuse-tumbleweed-20230415-DVD-x86_64-full-packaged-desktop-environments",
    "manjaro-kde-21.1.6-210917-linux514-full-hardware-support-edition",
    "elementaryos-6.1-odin-amd64-20230104-iso-with-pantheon-desktop-environment",
    "pop_os-21.10-amd64-nvidia-proprietary-drivers-included-live",
    "kali-linux-2023.2-live-amd64-iso-with-persistent-storage-and-custom-tools",
    "zorin-os-16-pro-ultimate-edition-64-bit-r1-iso-with-windows-app-support",
    "endeavouros-2023.04.15-x86_64-iso-with-offline-installer-and-xfce4",
    "mx-linux-21.2-aarch64-xfce-iso-with-ahs-enabled-kernel-and-snapshot-feature",
    "solus-4.3-budgie-desktop-environment-full-iso-with-software-center",
    "slackware-15.0-install-dvd-iso-with-extended-documentation-and-extras",
    "alpine-standard-3.15.0-x86_64-iso-for-container-and-server-use",
    "gentoo-livecd-amd64-minimal-20230407-stage3-tarball-included",
    "peppermint-11-20210903-amd64-iso-with-hybrid-lxde-xfce-desktop",
    "deepin-20.3-amd64-iso-with-deepin-desktop-environment-and-app-store"
  ];
  
  return Array.from({ length: count }, () => linuxIsos[Math.floor(Math.random() * linuxIsos.length)]);
};
  