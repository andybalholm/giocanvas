#!/bin/bash
for i in $(cat cl)
do
	cd $i
	echo -n "$i "
	case $i in
		showimage)
			./showimage showimage.png & 
			;;
		elections)
			./elections nyt-????.d &
			;;
		*)
			./$i &
			;;
	esac
	cd ..
done
echo
