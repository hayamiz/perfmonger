AC_PREREQ(2.59)

AC_INIT(perfmonger, 0.5.0-dev, y.hayamizu@gmail.com)
AC_CONFIG_AUX_DIR([config])
AC_CONFIG_HEADER([config.h])
AC_CONFIG_MACRO_DIR([m4])

AM_INIT_AUTOMAKE([check-news foreign 1.11 -Wall -Wno-portability silent-rules tar-pax subdir-objects])
AC_SUBST(PACKAGE_VERSION)

AC_PROG_CC_STDC
AC_PROG_LIBTOOL
AM_SILENT_RULES([yes])
AC_PREFIX_DEFAULT([/usr/local])

AC_CONFIG_FILES([Makefile
	src/Makefile
	src/sysstat/Makefile
	src/ruby/Makefile
	src/ruby/perfmonger/Makefile
	src/ruby/perfmonger/command/Makefile
	data/Makefile
	data/assets/Makefile
	data/assets/js/Makefile
	data/assets/css/Makefile
	data/assets/img/Makefile
	package/Makefile
	package/rpm/Makefile
	test/Makefile])
AC_CONFIG_FILES([src/sysstat/sysconfig.h:src/sysstat/sysconfig.in
	src/ruby/perfmonger/version.rb
	package/rpm/perfmonger.spec])


dnl **************************************************************
dnl Check for Ruby
dnl **************************************************************
ruby_available="no"
AC_ARG_WITH([ruby],
            AS_HELP_STRING([--with-ruby=PATH],
                           [Ruby interpreter path (default: auto-detect)]),
            [RUBY="$withval"])

if test "$RUBY" = "no"; then
  : # ignore
elif test "$RUBY" = ""; then
  AC_PATH_PROG(RUBY, ruby, none)
  if test "$RUBY" != "none"; then
    ruby_available="yes"
  fi
else
  AC_CHECK_FILE([$RUBY],
                [ruby_available="yes"],
                [AC_MSG_ERROR([$RUBY is not found.])])
fi

if test "x$ruby_available" != "xyes"; then
  AC_MSG_ERROR([Ruby is required.])
fi

perfmonger_ruby_path=$RUBY
AC_SUBST(perfmonger_ruby_path)

dnl **************************************************************
dnl Check for Cutter
dnl **************************************************************

m4_ifdef([AC_CHECK_GCUTTER],
         [AC_CHECK_GCUTTER],
         [cutter_use_cutter="no"])

AM_CONDITIONAL([WITH_CUTTER], [test "$cutter_use_cutter" != "no"])
if test "$cutter_use_cutter" = "yes"; then
  AC_DEFINE(WITH_CUTTER, 1, [Define to 1 if you use Cutter])
fi


dnl **************************************************************
dnl Check for path
dnl **************************************************************

perfmonger_libdir="$libdir/perfmonger"
perfmonger_rubylibdir="$perfmonger_libdir/ruby"
perfmonger_datarootdir="$datarootdir/perfmonger"

AC_SUBST(perfmonger_rubylibdir)
AC_SUBST(perfmonger_datarootdir)


dnl **************************************************************
dnl Misc
dnl **************************************************************

perfmonger_ac_date=`date +'%Y-%m-%d %H:%M:%S'`
AC_SUBST(perfmonger_ac_date)

perfmonger_build_aux=""
if test -d '.git'; then
  perfmonger_build_aux=", git:$(git log | head -1 | awk '{print $2}' | head -c 8)"
fi

AC_SUBST(perfmonger_build_aux)

AC_OUTPUT


echo
echo "Configure Result:"
echo "== Prerequisites =="
echo "  Ruby : $RUBY"
echo " GLib2 : $ac_glib_2_0_available"
echo ""
echo "== Optional requirements =="
echo " Cutter(optional) : $CUTTER"
echo ""
echo
