# .github/assets/demo.tape: regenerates demo.gif.
# Run from the repo root: vhs .github/assets/demo.tape
# Needs vhs, ttyd, ffmpeg, Go and an ssh client. It builds the server and runs it
# on 127.0.0.1:2299 with a throwaway host key, so nothing touches the live box.

Output .github/assets/demo.gif

Require go
Require ssh

Set Shell bash
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 18
Set Width 1200
Set Height 720
Set Padding 24
Set Theme "TokyoNight"
Set TypingSpeed 60ms

# Build and start a local server, connect, and let the boot animation finish.
Hide
Type "export PS1='$ ' D=$(mktemp -d) && go build -o $D/srv ./cmd/server && (LISTEN_ADDR=127.0.0.1:2299 HOST_KEY_PATH=$D/key $D/srv >/dev/null 2>&1 &) && sleep 1 && clear"
Enter
Sleep 15s
Type "ssh -p 2299 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR localhost"
Enter
Sleep 4s
Show

# Home card -> Projects -> Kessler -> back home.
Sleep 1.5s
Down
Sleep 400ms
Down
Sleep 700ms
Enter
Sleep 1.5s
Enter
Sleep 2.5s
Escape
Sleep 1s
Escape
Sleep 700ms
Up
Sleep 400ms
Up
Sleep 1.5s

# Leave and stop the server.
Hide
Type "q"
Sleep 500ms
Type "pkill -f $D/srv; rm -rf $D"
Enter
Sleep 500ms
