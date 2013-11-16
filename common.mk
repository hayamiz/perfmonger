edit = sed					\
	-e 's|@bindir[@]|$(bindir)|g'		\
	-e 's|@pkgdatadir[@]|$(pkgdatadir)|g'	\
	-e 's|@prefix[@]|$(prefix)|g'		\
	-e 's|@perfmonger_ac_date[@]|$(perfmonger_ac_date)|g'		\
	-e 's|@configure_input[@]|$(configure_input)|g'		\
	-e 's|@perfmonger_libdir[@]|$(perfmonger_libdir)|g'		\
	-e 's|@perfmonger_rubylibdir[@]|$(perfmonger_rubylibdir)|g'	\
	-e 's|@perfmonger_datarootdir[@]|$(perfmonger_datarootdir)|g'	\
	-e 's|@perfmonger_ruby_path[@]|$(perfmonger_ruby_path)|g'
