package cmd

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"strconv"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/spf13/cobra"
)

var (
	checkBase string
	step      = 10240
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "check file info",
	Long:  `md5 and sha1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		orginFile, err := os.OpenFile(checkBase, os.O_RDONLY, 0666)
		if err != nil {
			return err
		}
		defer orginFile.Close()

		fileInfo, err := orginFile.Stat()
		if err != nil {
			return err
		}

		filesize := fileInfo.Size()

		// log beatufy
		if err := ui.Init(); err != nil {
			return err
		}

		g := widgets.NewGauge()
		g.SetRect(0, 3, 50, 6)
		g.BarColor = ui.ColorYellow
		g.LabelStyle = ui.NewStyle(ui.ColorBlue)
		g.BorderStyle.Fg = ui.ColorWhite

		written := 0 // 总读取大小
		bs := make([]byte, step)
		_md5 := md5.New()
		_sha1 := sha1.New()
		for {
			n, err := orginFile.Read(bs)
			if err != nil {
				if err == io.EOF {
					break
				}

				return err
			}

			bsslice := bs[:n]
			_md5.Write(bsslice)
			_sha1.Write(bsslice)

			written = written + len(bsslice)

			g.Title = fmt.Sprintf("calculating: %d/%d", written, filesize)
			g.Percent = int(float64(written) / float64(filesize) * 100)
			ui.Render(g)
		}
		ui.Close()

		m := float64(written) / float64(1024*1024)
		fmt.Println("read file size: " + strconv.FormatFloat(m, 'f', 2, 64) + "M(step by " + strconv.Itoa(step) + ")")

		fmt.Printf("[ md5]: %X\n", _md5.Sum(nil))
		fmt.Printf("[sha1]: %X\n", _sha1.Sum(nil))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// checkCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	checkCmd.Flags().StringVarP(&checkBase, "path", "p", "./demo", "file path")
}
