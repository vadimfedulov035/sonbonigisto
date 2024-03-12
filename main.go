package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	oggvorbis "github.com/jfreymuth/oggvorbis"
)

// panic errors
const ERROR = "%s does not exist. Ensure Audacity is running and mod-script-pipe is set to Enabled in the Preferences window."
const CmdFailedErr = "Failed!"
const AudacityFallErr = "Fall!"
// temporary directory
const PATH = "/root/sonbonigisto/.tmp/"
// commands to Audacity
const ImportCmd = "Import2: Filename=\"%s\""
const CompressCmd = "Compressor: Threshold=\"-27.5\" NoiseFloor=\"-30.0\" Ratio=\"3.0\" AttackTime=\"0.2\" ReleaseTime=\"1.0\" Normalize=\"True\" UsePeak=\"False\""
const TruncSilenceCmd = "TruncateSilence: Threshold=\"-27.5\" Action=\"Compress Excess Silence\" Minimum=\"0.1\" Truncate=\"0.1\" Compress=\"33\" Independent=\"False\""
const NormalizeCmd = "Normalize: PeakLevel=\"-5\" ApplyGain=\"True\" RemoveDcOffset=\"True\" StereoIndependent=\"False\""
const GraphEQCmd = "GraphicEq: FilterLength=\"8191\" InterpolateLin=\"False\" InterpolationMethod=\"B-spline\" f0=\"62.77682\" f1=\"70.002037\" f2=\"73.252718\" f3=\"83.938837\" f4=\"97.946153\" f5=\"119.59827\" f6=\"152.81833\" f7=\"195.26572\" f8=\"221.72906\" f9=\"256.39197\" f10=\"336.65327\" f11=\"492.91565\" f12=\"591.0509\" f13=\"689.68267\" f14=\"776.07428\" f15=\"889.28813\" f16=\"982.67802\" f17=\"9948.9608\" f18=\"11195.196\" f19=\"12597.538\" f20=\"14047.435\" f21=\"16096.677\" v0=\"-41.952381\" v1=\"-10.904762\" v2=\"-5.2349215\" v3=\"-1.6888895\" v4=\"0.13650751\" v5=\"1.1746035\" v6=\"2\" v7=\"2\" v8=\"1.3015871\" v9=\"0.031745911\" v10=\"0.031745911\" v11=\"-2.984127\" v12=\"-1.3650789\" v13=\"-0.015872955\" v14=\"0.469841\" v15=\"0.469841\" v16=\"-0.015872955\" v17=\"-0.015872955\" v18=\"-0.031746864\" v19=\"-0.031746864\" v20=\"-1.7301588\" v21=\"-41.885715\""
const ExportCmd = "Export2: Filename=\"%s\" NumChannels=\"2\""
const TrackCloseCmd = "TrackClose:"
const CloseCmd = "Close:"
// compact durations
const SecondWait = 1 * time.Second
const FiveSecondsWait = 5 * time.Second
const TenSecondsWait = 10 * time.Second

// loads initialization config
func loadInitConfig(filename string) string {
    type InitConfig struct {
        KeyAPI string
    }
    var initConfig InitConfig
    data, err := os.ReadFile(filename)
    if err != nil {
        panic(err)
    }
    json.Unmarshal(data, &initConfig)
    return initConfig.KeyAPI
}

// sets default pipe filenames for Linux
func setPipeNames() (string, string, rune) {
	writePipeName := fmt.Sprintf("/tmp/audacity_script_pipe.to.%d", os.Getuid())
	readPipeName := fmt.Sprintf("/tmp/audacity_script_pipe.from.%d", os.Getuid())
	var eol rune = '\n'
	return writePipeName, readPipeName, eol
}

// checks if pipes exist
func checkPipes(writePipeName string, readPipeName string) {
	if _, err := os.Stat(writePipeName); os.IsNotExist(err) {
		panic(fmt.Sprintf(ERROR, writePipeName))
	}
	if _, err := os.Stat(readPipeName); os.IsNotExist(err) {
		panic(fmt.Sprintf(ERROR, readPipeName))
	}
}

// passes command to pipe
func passToPipe(command string, writePipeName string, readPipeName string, eol rune) (string, error) {
	// open write pipe file
	writePipe, err := os.OpenFile(writePipeName, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		panic(err)
	}
	defer writePipe.Close()
	// open read pipe file
	readPipe, err := os.OpenFile(readPipeName, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		panic(err)
	}
	defer readPipe.Close()
	// write command to pipe
	writePipe.WriteString(command + string(eol))
	writePipe.Sync()
	// read response from pipe
	response := ""
	line := ""
	scanner := bufio.NewScanner(readPipe)
	for scanner.Scan() {
		line = scanner.Text()
		response += line
		if line == "" && len(response) > 0 {
			break
		}
	}
	// check if command finished successfully
	if strings.Contains(response, CmdFailedErr) {
		err := errors.New(CmdFailedErr)
		return response, err
	}
	return response, nil
}


// downloads voice message to path
func downloadVoiceMessage(filename string, bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// get voice message temporary file
	voiceID := msg.Voice.FileID
	fileConfig := tgbotapi.FileConfig{
		FileID: voiceID,
	}
	tmp, _ := bot.GetFile(fileConfig)
	fileID := tmp.FileID
	fileURL, _ := bot.GetFileDirectURL(fileID)
	// download data
	response, _ := http.Get(fileURL)
	defer response.Body.Close()
	// create file
	file, _ := os.Create(filename)
	defer file.Close()
	// copy data to file
	io.Copy(file, response.Body)

}

// calculates duration of audio file
func calcDuration(filename string) int {
	file, _ := os.Open(filename)
	defer file.Close()
	decoder, _ := oggvorbis.NewReader(file)
	sampleRate := decoder.SampleRate()
	numSamples := decoder.Length()
	duration := int(float64(numSamples) / float64(sampleRate))
	return duration
}

// checks if Audacity is active
func checkAudacity() error {
	var err error = nil
	processName := "audacity"
	out, err := exec.Command("pgrep", "-x", processName).Output()
	pid := strings.TrimSpace(string(out))
	if pid == "" {
		err = errors.New(AudacityFallErr)
	}
	return err
}

// kill Audacity
func killAudacity() {
	log.Println("Kill Audacity […]")
	defer log.Println("Kill Audacity [✓]")
	processName := "audacity"
	cmd := exec.Command("pkill", processName)
	cmd.Run()
	cmd.Wait()
}

// performs command via Audacity
func do(command string) error {
	if err := checkAudacity(); err != nil {
		return err
	}
	writePipeName, readPipeName, eol := setPipeNames()
	if err := checkAudacity(); err != nil {
		return err
	}
	checkPipes(writePipeName, readPipeName)
	if err := checkAudacity(); err != nil {
		return err
	}
	if _, err := passToPipe(command, writePipeName, readPipeName, eol); err != nil {
		return err
	}
	if err := checkAudacity(); err != nil {
		return err
	}
	return nil
}

// improves sound via Audacity
func improveSound(inputPath string, outputPath string) error {
	// import, compress, truncate silence, normalize, export, save, close
	if err := do(fmt.Sprintf(ImportCmd, inputPath)); err != nil {
		return err
	}
	if err := do(CompressCmd); err != nil {
		return err
	}
	if err := do(TruncSilenceCmd); err != nil {
		return err
	}
	if err := do(NormalizeCmd); err != nil {
		return err
	}
	if err := do(GraphEQCmd); err != nil {
		return err
	}
	if err := do(fmt.Sprintf(ExportCmd, outputPath)); err != nil {
		return err
	}
	if err := do(TrackCloseCmd); err != nil {
		return err
	}
	if err := do(CloseCmd); err != nil {
		return err
	}
	return nil
}

// handles all operations to voice message
func handleVoiceMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// log handling voice message
	log.Printf("Voice message [!]")
	defer log.Println("Voice message [✓]")
	// create and delete temporary directory
	os.MkdirAll(PATH, os.ModePerm)
	defer os.RemoveAll(PATH)
	// set paths
	inputPath := path.Join(PATH, "sono.ogg")
	outputPath := path.Join(PATH, "bonsono.ogg")
	// download voice message and improve its sound
	downloadVoiceMessage(inputPath, bot, msg)
	// try to improve sound until success
	success := true
	for {
		err := improveSound(inputPath, outputPath)
		// log reconnection after error
		if !success {
			if err != nil {
				log.Println("Reconnect to Audacity [x]")
			} else {
				log.Println("Reconnect to Audacity [✓]")
			}
		}
		// kill Audacity if no fall and wait until it restarts
		if err != nil {
			success = false
			log.Println("Audacity [x]:", err)
			if err.Error() == CmdFailedErr {
				killAudacity()
			}
			for {
				err = checkAudacity()
				if err == nil {
					time.Sleep(TenSecondsWait)
					log.Println("Audacity [✓]")
					break
				}
				time.Sleep(SecondWait)
			}
		} else {
			break
		}
	}
	// calculate new duration
	duration := calcDuration(outputPath)
	// resend better voice message
	audioConfig := tgbotapi.NewAudioUpload(msg.Chat.ID, outputPath)
	audioConfig.Duration = duration
	audioConfig.Caption = "#mezvenka"
	for {
		_, err := bot.Send(audioConfig)
		if err != nil {
			log.Println(err)
			time.Sleep(FiveSecondsWait)
		} else {
			break
		}
	}
}

// clean temporary directories
func cleanTmp() {
	tmpDir := [2]string{"/var/tmp/audacity-root/*", "/root/.audacity-data/AutoSave/*"}
	cmd := exec.Command("rm", "-rf", tmpDir[0], tmpDir[1])
	cmd.Run()
	cmd.Wait()
}

func main() {
	// initialize bot
	token := loadInitConfig("token.json")
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		panic(err)
	}
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 25
	updates, _ := bot.GetUpdatesChan(u)
	// every 60 seconds check for new message
	for update := range updates {
		// every valid voice message get info and handle it
		if update.Message.Voice != nil {
			msg := update.Message
			handleVoiceMessage(bot, msg)
			cleanTmp()
		}
	}
}
