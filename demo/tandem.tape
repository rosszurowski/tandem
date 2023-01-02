Set Shell fish
Set Width 1360
Set FontSize 20
Set Padding 40

Require php
Require yarn
Require tandem

Hide
Type "yarn install && clear"
Enter
Sleep 1s
Show

Sleep 1s
Type "tandem \"
Enter
Sleep 0.25s
Type@35ms "'php -S  localhost:8000' \"
Enter
Sleep 0.15s
Type@30ms "'esbuild src/*.ts --bundle --outdir=dist --watch' \"
Enter
Sleep 0.18s
Type@25ms "'tailwindcss -i src/index.css -o dist/index.css --watch'"
Sleep 0.5s
Enter
Sleep 8s
Ctrl+C
Sleep 4s

