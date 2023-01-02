Set Shell fish
Set Padding 40
Set Framerate 60
Set Theme "nord"

Require php
Require yarn
Require tandem

Hide
Type@1ms tput rmam
Enter
Type@1ms `function fish_prompt; echo -e "$(set_color 4C566A)\$ $(set_color normal)"; end`
Enter
Type@1ms "yarn install"
Enter
Type@1ms `./update.sh 11 'console.log("my-8 pl-2 pr-5 bg-red-500 text-white")' &; disown`
Enter
Ctrl+L
Sleep 0.2s
Show

Sleep 1s
Type "tandem \"
Enter
Sleep 0.15s
Type@24ms "'php -S localhost:8000' \"
Enter
Sleep 0.1s
Type@18ms "'esbuild src/*.ts --bundle --outdir=dist --watch' \"
Enter
Sleep 0.1s
Type@16ms "'tailwindcss -i src/index.css -o dist/index.css --watch'"
Sleep 0.5s
Enter

Sleep 7s
Ctrl+C
Sleep 3s
Ctrl+L

Hide
Type@1ms `./update.sh 0 'console.log("my-8 pl-2 pr-5 text-white");'`
Enter
