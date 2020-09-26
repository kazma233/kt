package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	excelize "github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/cobra"
)

type (
	// Font font style
	Font struct {
		Bold      bool    `json:"bold"`
		Italic    bool    `json:"italic"`
		Underline string  `json:"underline"`
		Family    string  `json:"family"`
		Size      float64 `json:"size"`
		Strike    bool    `json:"strike"`
		Color     string  `json:"color"`
	}

	// Fill fill style
	Fill struct {
		Type    string   `json:"type"`
		Pattern int      `json:"pattern"`
		Color   []string `json:"color"`
		Shading int      `json:"shading"`
	}

	// Border border style
	Border struct {
		Type  string `json:"type"`
		Color string `json:"color"`
		Style int    `json:"style"`
	}

	// Style style
	Style struct {
		Border []Border `json:"border"`
		Fill   Fill     `json:"fill"`
		Font   *Font    `json:"font"`
	}

	// Header excel header
	Header struct {
		Name    string
		Comment string
	}

	// FieldInfo excel field list struct
	FieldInfo struct {
		Name    string
		Type    string
		Key     string
		Null    string
		Default string
		Comment string
	}

	// SecTableInfo table info
	SecTableInfo struct {
		TableName    string         `db:"TABLE_NAME"`
		TableComment sql.NullString `db:"TABLE_COMMENT"`
	}

	// SecFieldInfo field info
	SecFieldInfo struct {
		FieldName string         `db:"COLUMN_NAME"`
		Type      string         `db:"COLUMN_TYPE"`
		Key       sql.NullString `db:"COLUMN_KEY"`
		Null      sql.NullString `db:"IS_NULLABLE"`
		Default   sql.NullString `db:"COLUMN_DEFAULT"`
		Comment   sql.NullString `db:"COLUMN_COMMENT"`
	}
)

var (
	// excel file
	excelFile *excelize.File
	// excel sheet control
	indexSheetName = "index"
	tableSheetName = "tabel"
	indexSheetNum  = 0
	tableSheetNum  = 1
	// excel line control
	indexSheetIndex = 1
	tableSheetIndex = 1
)

var (
	host     string
	username string
	password string
	port     int
	schame   string
)

// db2excelCmd represents the db2excel command
var db2excelCmd = &cobra.Command{
	Use:   "db2excel",
	Short: "export mysql database struct to excel",
	Long:  `'kt db2excel -h' show usage`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if password == "" {
			fmt.Printf("password: ")
			_, err := fmt.Scanln(&password)
			if err != nil {
				return err
			}
		}

		db, err := sqlx.Open("mysql",
			fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", username, password, host, port, schame),
		)
		if err != nil {
			return err
		}

		tableInfos := GetDMLTableInfo(db, schame)
		for _, tableInfo := range tableInfos {
			header := &Header{
				Name:    tableInfo.TableName,
				Comment: tableInfo.TableComment.String,
			}
			WriteIndexSheet(header)

			filedInfos := GetDMLFieldInfo(db, schame, tableInfo.TableName)
			excelFieldInfos := []FieldInfo{}
			for _, filedInfo := range filedInfos {
				excelFieldInfos = append(excelFieldInfos, FieldInfo{
					Name:    filedInfo.FieldName,
					Type:    filedInfo.Type,
					Key:     filedInfo.Key.String,
					Null:    filedInfo.Null.String,
					Default: filedInfo.Default.String,
					Comment: filedInfo.Comment.String,
				})
			}

			WriteTableInfo(header, excelFieldInfos)
		}

		Save(schame)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(db2excelCmd)

	excelFile = excelize.NewFile()

	// "index" sheet setting
	excelFile.SetSheetName(excelFile.GetSheetName(0), indexSheetName)
	excelFile.SetColWidth(indexSheetName, "A", "F", 20)

	// "table" sheet settng
	excelFile.SetActiveSheet(excelFile.NewSheet(tableSheetName))
	excelFile.SetColWidth(tableSheetName, "A", "A", 40)
	excelFile.SetColWidth(tableSheetName, "B", "E", 20)
	excelFile.SetColWidth(tableSheetName, "F", "F", 40)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// db2excelCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	db2excelCmd.Flags().StringVarP(&host, "host", "", "127.0.0.1", "host")
	db2excelCmd.Flags().StringVarP(&username, "username", "u", "root", "username")
	db2excelCmd.Flags().IntVarP(&port, "port", "p", 3306, "port. must a int type")
	db2excelCmd.Flags().StringVarP(&schame, "schame", "s", "mysql", "export schame name")
	db2excelCmd.Flags().StringVarP(&password, "password", "", "", "password")
}

// GetDMLTableInfo get dml table info
func GetDMLTableInfo(db *sqlx.DB, dbName string) []SecTableInfo {
	tableInfos := []SecTableInfo{}
	if err := db.Select(&tableInfos,
		"SELECT TABLE_NAME, TABLE_COMMENT from information_schema.TABLES where TABLE_SCHEMA = '"+dbName+"' and TABLE_TYPE = 'BASE TABLE'",
	); err != nil {
		panic(err)
	}

	log.Printf("get %v table", len(tableInfos))

	return tableInfos
}

// GetDMLFieldInfo get dml field infos
func GetDMLFieldInfo(db *sqlx.DB, dnName, tableName string) []SecFieldInfo {
	fieldInfos := []SecFieldInfo{}
	if err := db.Select(&fieldInfos, "SELECT COLUMN_NAME, IS_NULLABLE, COLUMN_TYPE, COLUMN_KEY, COLUMN_DEFAULT, COLUMN_COMMENT "+
		"from information_schema.`COLUMNS` "+
		"where TABLE_NAME = '"+tableName+"' and TABLE_SCHEMA = '"+dnName+"' order by ORDINAL_POSITION",
	); err != nil {
		panic(err)
	}

	log.Printf("%s table has %v field", tableName, len(fieldInfos))

	return fieldInfos
}

// WriteIndexSheet write index sheet
func WriteIndexSheet(excelHeader *Header) {
	excelFile.SetActiveSheet(excelFile.GetSheetIndex(indexSheetName))

	currentIndexStr := strconv.Itoa(indexSheetIndex)
	writeHeader(indexSheetName, currentIndexStr, excelHeader)
	indexSheetIndex++
}

// WriteTableInfo write a table info
func WriteTableInfo(excelHeader *Header, fieldInfos []FieldInfo) {
	excelFile.SetActiveSheet(excelFile.GetSheetIndex(tableSheetName))

	// write header
	tableSheetIndexStr := strconv.Itoa(tableSheetIndex)
	writeHeader(tableSheetName, tableSheetIndexStr, excelHeader)
	// setting style
	excelFile.SetCellStyle(tableSheetName, "A"+tableSheetIndexStr, "F"+tableSheetIndexStr, getHeaderStyleString(excelFile))
	// setting link
	excelFile.SetCellHyperLink(indexSheetName, "A"+strconv.Itoa(indexSheetIndex-1), tableSheetName+"!"+"A"+tableSheetIndexStr+":F"+tableSheetIndexStr, "Location")
	tableSheetIndex++

	// write section
	tableSheetIndexStr = strconv.Itoa(tableSheetIndex)
	hcell := "A" + tableSheetIndexStr
	excelFile.SetCellStr(tableSheetName, "A"+tableSheetIndexStr, "字段名称")
	excelFile.SetCellStr(tableSheetName, "B"+tableSheetIndexStr, "字段类型")
	excelFile.SetCellStr(tableSheetName, "C"+tableSheetIndexStr, "键")
	excelFile.SetCellStr(tableSheetName, "D"+tableSheetIndexStr, "是否允许为空")
	excelFile.SetCellStr(tableSheetName, "E"+tableSheetIndexStr, "默认值")
	excelFile.SetCellStr(tableSheetName, "F"+tableSheetIndexStr, "注释")
	tableSheetIndex++

	// write body
	for _, filedInfo := range fieldInfos {
		tableSheetIndex = writeOneLineExcel(excelFile, tableSheetName, tableSheetIndex, &filedInfo)
	}

	// set boder style
	vcell := "F" + strconv.Itoa(tableSheetIndex-1)
	excelFile.SetCellStyle(tableSheetName, hcell, vcell, getBorderStyleString(excelFile))
	tableSheetIndex++
}

// Save save excel file
func Save(fileName string) {
	tableSheetIndex = 1
	excelFile.SetActiveSheet(excelFile.GetSheetIndex(indexSheetName))
	excelFile.SaveAs(fileName + ".xlsx")
}

func writeHeader(sheetName, currentIndexStr string, val *Header) {
	excelFile.MergeCell(sheetName, "A"+currentIndexStr, "C"+currentIndexStr)
	excelFile.SetCellStr(sheetName, "A"+currentIndexStr, val.Name)
	excelFile.MergeCell(sheetName, "D"+currentIndexStr, "F"+currentIndexStr)
	excelFile.SetCellStr(sheetName, "D"+currentIndexStr, val.Comment)
}

func writeOneLineExcel(excelFile *excelize.File, sheetName string, postion int, filedInfo *FieldInfo) int {
	currentPostion := strconv.Itoa(postion)

	excelFile.SetCellStr(sheetName, "A"+currentPostion, filedInfo.Name)
	excelFile.SetCellStr(sheetName, "B"+currentPostion, filedInfo.Type)
	excelFile.SetCellStr(sheetName, "C"+currentPostion, filedInfo.Key)
	excelFile.SetCellStr(sheetName, "D"+currentPostion, filedInfo.Null)
	excelFile.SetCellStr(sheetName, "E"+currentPostion, filedInfo.Default)
	excelFile.SetCellStr(sheetName, "F"+currentPostion, filedInfo.Comment)

	return postion + 1
}

// boder style
func getBorderStyleString(excelFile *excelize.File) int {
	bs := Style{
		Border: []Border{
			{
				Type:  "left",
				Color: "000000",
				Style: 1,
			}, {
				Type:  "top",
				Color: "000000",
				Style: 1,
			}, {
				Type:  "bottom",
				Color: "000000",
				Style: 1,
			}, {
				Type:  "right",
				Color: "000000",
				Style: 1,
			},
		},
	}

	bsByte, _ := json.Marshal(bs)
	borderStyle, _ := excelFile.NewStyle(string(bsByte))
	return borderStyle
}

// header style
func getHeaderStyleString(excelFile *excelize.File) int {
	style := Style{
		Border: []Border{
			{
				Type:  "left",
				Color: "000000",
				Style: 1,
			}, {
				Type:  "top",
				Color: "000000",
				Style: 1,
			}, {
				Type:  "bottom",
				Color: "000000",
				Style: 1,
			}, {
				Type:  "right",
				Color: "000000",
				Style: 1,
			},
		},
		Fill: Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"009999"},
		},
		Font: &Font{
			Bold:  true,
			Color: "FFFFFF",
		},
	}

	styleByte, _ := json.Marshal(style)
	styleVal, _ := excelFile.NewStyle(string(styleByte))
	return styleVal
}
