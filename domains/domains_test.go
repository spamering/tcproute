package domains

import (
	"testing"
)

func TestDomainType(t *testing.T) {
	v,err:=ParseDomainType(" bAse \r\n")
	if err != nil||  v!=Base{
		t.Error(err)
	}

	if v.String()!="Base"{
		t.Error(`v.String()!="Base"`)
	}


	v,err=ParseDomainType(" 010101 \r\n")
	if err == nil || v!=DomainType(0){
		t.Error(err)
	}

	if v.String()!="Unknown"{
		t.Error(v.String())
	}

	v,err=ParseDomainType(" SuFfix \r\n")
	if err != nil||  v!=DomainType(2){
		t.Error(err)
	}
}


func TestDomains(t *testing.T) {

	d := NewDomains(100)

	add := func(domain string, domainType DomainType, userdata UserData) {
		if err := d.Add(domain, domainType, userdata); err != nil {
			t.Fatal(err)
		}
	}

	add("163.com", Base, "1.163.com")
	add("163.com", Base, "2.163.com")
	add("baidu.com", Suffix, "1.baidu.com")
	add("baidu.com", Suffix, "2.baidu.com")
	add("*.qq.com", Pan, "1.qq.com")
	add("*.qq.com", Pan, "2.qq.com")
	add("ww?.google.com", Pan, "1.google.com")
	add("ww?.google.com", Pan, "2.google.com")
	add(`^www\..+?\.com$`, Regex, "1.xxx.com")
	add(`^www\..+?\.com$`, Regex, "2.xxx.com")

	// base 测试
	if res := d.Find("163.com"); len(res.Userdatas) != 2 {
		t.Error(res.Userdatas)
	}else {
		for _, v := range res.Userdatas {
			str := v.(string)
			if str != "1.163.com" && str != "2.163.com" {
				t.Error(str)
			}
		}
	}
	if res := d.Find("www.163.net"); len(res.Userdatas) != 0 {
		t.Error(res.Userdatas)
	}
	if res := d.Find("abc.163.com"); len(res.Userdatas) != 0 {
		t.Error(res.Userdatas)
	}

	//后缀测试
	if res := d.Find("abc.baidu.com"); len(res.Userdatas) != 2 {
		t.Error(res.Userdatas)
	} else {
		for _, v := range res.Userdatas {
			str := v.(string)
			if str != "1.baidu.com" && str != "2.baidu.com" {
				t.Error(str)
			}
		}
	}
	if res := d.Find("baidu.com"); len(res.Userdatas) != 2 {
		t.Error(res.Userdatas)
	} else {
		for _, v := range res.Userdatas {
			str := v.(string)
			if str != "1.baidu.com" && str != "2.baidu.com" {
				t.Error(str)
			}
		}
	}

	//后缀测试、正则测试
	if res := d.Find("abc.qq.com"); len(res.Userdatas) != 2 {
		t.Error(res.Userdatas)
	} else {
		for _, v := range res.Userdatas {
			str := v.(string)
			if str != "1.qq.com" && str != "2.qq.com" {
				t.Error(str)
			}
		}
	}

	//*泛解析测试
	if res := d.Find("aaa.qq.com"); len(res.Userdatas) != 2 {
		t.Error(res.Userdatas)
	} else {
		for _, v := range res.Userdatas {
			str := v.(string)
			if str != "1.qq.com" && str != "2.qq.com" {
				t.Error(str)
			}
		}
	}
	if res := d.Find("qq.com"); len(res.Userdatas) != 0 {
		t.Error(res.Userdatas)
	}

	//?泛解析测试
	if res := d.Find("wwa.google.com"); len(res.Userdatas) != 2 {
		t.Error(res.Userdatas)
	} else {
		for _, v := range res.Userdatas {
			str := v.(string)
			if str != "1.google.com" && str != "2.google.com" {
				t.Error(str)
			}
		}
	}
	if res := d.Find("ww.google.com"); len(res.Userdatas) != 0 {
		t.Error(res.Userdatas)
	}


	//正则测试
	if res := d.Find("www.abc.com"); len(res.Userdatas) != 2 {
		t.Error(res.Userdatas)
	} else {
		for _, v := range res.Userdatas {
			str := v.(string)
			if str != "1.xxx.com" && str != "2.xxx.com" {
				t.Error(str)
			}
		}
	}

	// Base 删除测试
	d.RemoveDomain("163.com", Base, func(domain string, domainType DomainType, uesrdata UserData) bool {
		if domain != "163.com" || domainType != Base {
			t.Error(`domain!="163.com"||domainType!=Base`)
		}
		if uesrdata.(string) == "1.163.com" {
			return true
		}
		return false
	})
	if res := d.Find("163.com"); len(res.Userdatas) != 1 {
		t.Error(res.Userdatas)
	}else {
		for _, v := range res.Userdatas {
			str := v.(string)
			if str != "2.163.com" {
				t.Error(str)
			}
		}
	}

	// Suffic 删除测试
	d.RemoveDomain("baidu.com", Suffix, func(domain string, domainType DomainType, uesrdata UserData) bool {
		if domain != "baidu.com" || domainType != Suffix {
			t.Error(`domain!="baidu.com"||domainType!=Suffix`)
		}
		if uesrdata.(string) == "1.baidu.com" {
			return true
		}
		return false
	})
	if res := d.Find("baidu.com"); len(res.Userdatas) != 1 {
		t.Error(res.Userdatas)
	}else {
		for _, v := range res.Userdatas {
			str := v.(string)
			if str != "2.baidu.com" {
				t.Error(str)
			}
		}
	}

	// Pan 删除测试 1
	d.RemoveDomain("*.qq.com", Pan, func(domain string, domainType DomainType, uesrdata UserData) bool {
		if domain != "*.qq.com" || domainType != Pan {
			t.Error(`domain!="*.qq.com"||domainType!=Pan`)
		}
		if uesrdata.(string) == "1.qq.com" {
			return true
		}
		return false
	})
	if res := d.Find("abc.qq.com"); len(res.Userdatas) != 1 {
		t.Error(res.Userdatas)
	}else {
		for _, v := range res.Userdatas {
			str := v.(string)
			if str != "2.qq.com" {
				t.Error(str)
			}
		}
	}

	// Pan 删除测试 2
	d.RemoveDomain("ww?.google.com", Pan, func(domain string, domainType DomainType, uesrdata UserData) bool {
		if domain != "ww?.google.com" || domainType != Pan {
			t.Error(`domain!="*.qq.com"||domainType!=Pan`)
		}
		if uesrdata.(string) == "1.google.com" {
			return true
		}
		return false
	})
	if res := d.Find("wwx.google.com"); len(res.Userdatas) != 1 {
		t.Error(res.Userdatas)
	}else {
		for _, v := range res.Userdatas {
			str := v.(string)
			if str != "2.google.com" {
				t.Error(str)
			}
		}
	}

	// 正则删除测试
	d.RemoveDomain(`^www\..+?\.com$`, Regex, func(domain string, domainType DomainType, uesrdata UserData) bool {
		if domain != `^www\..+?\.com$` || domainType != Regex {
			t.Error(`domain!="^www\..+?\.com$"||domainType!=Regex`)
		}
		if uesrdata.(string) == "1.xxx.com" {
			return true
		}
		return false
	})
	if res := d.Find("www.xxx.com"); len(res.Userdatas) != 1 {
		t.Error(res.Userdatas)
	}else {
		for _, v := range res.Userdatas {
			str := v.(string)
			if str != "2.xxx.com" {
				t.Error(str)
			}
		}
	}

	//全部清空测试
	d.Remove(func(domain string, domainType DomainType, uesrdata UserData) bool {
		str := uesrdata.(string)
		if str != "2.163.com" && str != "2.baidu.com" && str != "2.qq.com" && str != "2.google.com" && str != "2.xxx.com" {
			t.Error(str)
		}
		return true
	})

	if res := d.Find("163.com"); len(res.Userdatas) != 0 {
		t.Error(res.Userdatas)
	}
	if res := d.Find("www.baidu.com"); len(res.Userdatas) != 0 {
		t.Error(res.Userdatas)
	}
	if res := d.Find("www.qq.com"); len(res.Userdatas) != 0 {
		t.Error(res.Userdatas)
	}
	if res := d.Find("www.google.com"); len(res.Userdatas) != 0 {
		t.Error(res.Userdatas)
	}
	if res := d.Find("www.xxx.com"); len(res.Userdatas) != 0 {
		t.Error(res.Userdatas)
	}

}