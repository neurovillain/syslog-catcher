package parser

// Допустимые типы данных полей сообщения.
// указаны в качестве примера, количество и значение должны совпадать
// с массивом fileKeyword.
const (
	plainText = iota
	deviceAddr
	devicePort
	portSpeed
	portDuplex
)

var (
	// Ключевые слова (имена полей) шаблона.
	fieldKeyword = []string{
		"$ignore$",
		"$device_addr$",
		"$device_port$",
		"$port_speed$",
		"$port_duplex$",
	}
)

// newTextField - создать новое поле на основе данных шаблона.
func newTextField(text string) *textField {
	for k, v := range fieldKeyword {
		if text == v {
			return &textField{
				dataType: k,
			}
		}
	}

	return &textField{
		text: text,
	}
}

// textField - поле (слово) текстового шаблона.
type textField struct {
	dataType int    // тип данных (0 - plainText)
	text     string // текст для сверки (только в случае если dataType != 0)
}

// match - сверить слово с тексовым шаблоном - в случае если
// тип данных текущего поля не plainText - возращает текущий тип данных
// для дальнейшей обработки, если поле содержит обычный текст -
// 0 - если текст совпадает и -1 если не совпадает.
func (f *textField) match(text string) int {
	if f.dataType != plainText {
		return f.dataType
	}
	if f.text == text {
		return 0
	}

	return -1
}
