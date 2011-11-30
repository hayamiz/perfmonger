AC_PREREQ(2.59)

AC_INIT(perfmonger, 0.0.0, y.hayamizu@gmail.com)
AC_CONFIG_AUX_DIR([config])
AC_CONFIG_HEADER([config.h])
AC_CONFIG_MACRO_DIR([m4])

AM_INIT_AUTOMAKE($PACKAGE_NAME, $PACKAGE_VERSION)

AC_PROG_LIBTOOL

AC_CONFIG_FILES([Makefile
	src/Makefile
	src/sysstat/Makefile
	test/Makefile])
AC_CONFIG_FILES([src/sysstat/sysconfig.h:src/sysstat/sysconfig.in])

dnl **************************************************************
dnl Sysstat settings
dnl **************************************************************

AC_MSG_CHECKING(sysstat configuration directory)
AC_ARG_VAR([sysstat_conf_dir],[sysstat configuration directory])
if test x$sysstat_conf_dir != x""; then
  SYSCONFIG_DIR=$sysstat_conf_dir
else
  if test -f /etc/sysconfig/sysstat.ioconf; then
    SYSCONFIG_DIR=/etc/sysconfig
  else
    SYSCONFIG_DIR=/etc/sysstat
  fi
fi
if test ! -f $SYSCONFIG_DIR/sysstat.ioconf; then
  AC_MSG_ERROR([sysstat.ioconf not found. Check "sysstat" is installed in your system.])
fi
AC_MSG_RESULT([$SYSCONFIG_DIR])
AC_SUBST(SYSCONFIG_DIR)


dnl **************************************************************
dnl Check for GLib
dnl **************************************************************

PKG_CHECK_MODULES([GLIB],
                  [glib-2.0 gthread-2.0],
                  [ac_glib_2_0_available="yes"],
                  [ac_glib_2_0_available="no"])

if test "$ac_glib_2_0_available" = "no"; then
  AC_MSG_ERROR([GLib is required.])
fi
AM_CONDITIONAL([WITH_GLIB], [test "$ac_glib_2_0_available" = "yes"])

dnl **************************************************************
dnl Check for Cutter
dnl **************************************************************

m4_ifdef([AC_CHECK_GCUTTER],
         [AC_CHECK_GCUTTER],
         [ac_cv_cutter_available="no"])

AM_CONDITIONAL([WITH_CUTTER], [test "$ac_cv_cutter_available" != "no"])


AC_OUTPUT

dnl **************************************************************
dnl Check for Mockart
dnl **************************************************************

AC_CHECK_HEADERS(mockart.h, [have_mockart_h=yes], [have_mockart_h=no])
AC_CHECK_LIB(mockart, mockart_init, [have_libmockart=yes], [have_libmockart=no])
AM_CONDITIONAL([WITH_MOCKART], [test "$have_libmockart" = "yes" && test "$have_mockart_h" = "yes"])

if test "$have_libmockart" = "yes" && test "$have_mockart_h" = "yes"; then
  have_mockart=yes
else
  have_mockart=no
fi

echo
echo "Configure Result:"
echo " sysstat conf : $SYSCONFIG_DIR/sysstat.ioconf"
echo "       Cutter : $CUTTER"
echo "        GLib2 : $ac_glib_2_0_available"
echo "      Mockart : $have_mockart"
echo
