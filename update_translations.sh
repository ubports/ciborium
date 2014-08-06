#!/bin/sh

sources=$(find . -name '*.go' | xargs)
domain='ciborium'
pot_file=po/$domain.pot

xgettext -o $pot_file \
 --add-comments \
 --from-code=UTF-8 \
 --c++ --qt --add-comments=TRANSLATORS \
 --keyword=Gettext --keyword=tr --keyword=tr:1,2 --keyword=N_ \
 --package-name=$domain \
 --copyright-holder='Canonical Ltd.' \
 $sources

if [ "$1" = "--commit" ] && [ -n "$(bzr status $pot_file)" ]; then
    echo Commiting $pot_file
    bzr commit -m "Updated translation template" $pot_file
fi
