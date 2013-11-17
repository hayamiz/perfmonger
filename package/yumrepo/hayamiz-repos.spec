Summary: hayamiz.com RPM repository configuration
Name: hayamiz-repos
Version: 1.0.0
Release: 0
License: GPLv3+
URL: https://github.com/hayamiz/
Source: hayamiz-repos.tar.gz
Group: System Environment/Base
BuildRoot: %{_tmppath}/%{name}-%{version}-%{release}-%(%{__id_u} -n)
BuildArchitectures: noarch

%description
PerfMonger RPM repository configuration

%prep
%setup -c

%build

%install
%{__rm} -rf %{buildroot}

%{__install} -Dp -m0644 RPM-GPG-KEY-hayamiz %{buildroot}%{_sysconfdir}/pki/rpm-gpg/RPM-GPG-KEY-hayamiz

%{__install} -Dp -m0644 hayamiz.repo %{buildroot}%{_sysconfdir}/yum.repos.d/hayamiz.repo

%clean
%{__rm} -rf %{buildroot}

%post
rpm -q gpg-pubkey-981a94c0-49775d36 &>/dev/null || \
  rpm --import %{_sysconfdir}/pki/rpm-gpg/RPM-GPG-KEY-hayamiz

%files
%defattr(-, root, root, 0755)
%doc *
%pubkey RPM-GPG-KEY-hayamiz
%dir %{_sysconfdir}/yum.repos.d/
%config(noreplace) %{_sysconfdir}/yum.repos.d/hayamiz.repo
%dir %{_sysconfdir}/pki/rpm-gpg/
%{_sysconfdir}/pki/rpm-gpg/RPM-GPG-KEY-hayamiz

%changelog
* Sun Nov 17 2013 Yuto HAYAMIZU <y.hayamizu@gmail.com> - master
- (1.0.0-0)
- Initial package.
