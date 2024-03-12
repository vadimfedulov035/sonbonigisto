#!/bin/bash
# kill PulseAudio if it is running
if pulseaudio --check; then
	figlet "Kill PulseAudio"
	pulseaudio --kill
	sleep 1
fi
# start PulseAudio
figlet "Start PulseAudio"
pulseaudio --start
# start Audacity
figlet "Start Audacity"
while ! audacity
do
	figlet "Restart Audacity"
	sleep 1
done
