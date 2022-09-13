package huaweiads

var MccList = map[int]string{
	202: "gr", //Greece
	204: "nl", //Netherlands (Kingdom of the)
	206: "be", //Belgium
	208: "fr", //France
	212: "mc", //Monaco (Principality of)
	213: "ad", //Andorra (Principality of)
	214: "es", //Spain
	216: "hu", //Hungary (Republic of)
	218: "ba", //Bosnia and Herzegovina
	219: "hr", //Croatia (Republic of)
	220: "rs", //Serbia and Montenegro
	222: "it", //Italy
	225: "va", //Vatican City State
	226: "ro", //Romania
	228: "ch", //Switzerland (Confederation of)
	230: "cz", //Czech Republic
	231: "sk", //Slovak Republic
	232: "at", //Austria
	234: "gb", //United Kingdom of Great Britain and Northern Ireland
	235: "gb", //United Kingdom of Great Britain and Northern Ireland
	238: "dk", //Denmark
	240: "se", //Sweden
	242: "no", //Norway
	244: "fi", //Finland
	246: "lt", //Lithuania (Republic of)
	247: "lv", //Latvia (Republic of)
	248: "ee", //Estonia (Republic of)
	250: "ru", //Russian Federation
	255: "ua", //Ukraine
	257: "by", //Belarus (Republic of)
	259: "md", //Moldova (Republic of)
	260: "pl", //Poland (Republic of)
	262: "de", //Germany (Federal Republic of)
	266: "gi", //Gibraltar
	268: "pt", //Portugal
	270: "lu", //Luxembourg
	272: "ie", //Ireland
	274: "is", //Iceland
	276: "al", //Albania (Republic of)
	278: "mt", //Malta
	280: "cy", //Cyprus (Republic of)
	282: "ge", //Georgia
	283: "am", //Armenia (Republic of)
	284: "bg", //Bulgaria (Republic of)
	286: "tr", //Turkey
	288: "fo", //Faroe Islands
	289: "ge", //Abkhazia (Georgia)
	290: "gl", //Greenland (Denmark)
	292: "sm", //San Marino (Republic of)
	293: "si", //Slovenia (Republic of)
	294: "mk", //The Former Yugoslav Republic of Macedonia
	295: "li", //Liechtenstein (Principality of)
	297: "me", //Montenegro (Republic of)
	302: "ca", //Canada
	308: "pm", //Saint Pierre and Miquelon (Collectivit territoriale de la Rpublique franaise)
	310: "us", //United States of America
	311: "us", //United States of America
	312: "us", //United States of America
	313: "us", //United States of America
	314: "us", //United States of America
	315: "us", //United States of America
	316: "us", //United States of America
	330: "pr", //Puerto Rico
	332: "vi", //United States Virgin Islands
	334: "mx", //Mexico
	338: "jm", //Jamaica
	340: "gp", //Guadeloupe (French Department of)
	342: "bb", //Barbados
	344: "ag", //Antigua and Barbuda
	346: "ky", //Cayman Islands
	348: "vg", //British Virgin Islands
	350: "bm", //Bermuda
	352: "gd", //Grenada
	354: "ms", //Montserrat
	356: "kn", //Saint Kitts and Nevis
	358: "lc", //Saint Lucia
	360: "vc", //Saint Vincent and the Grenadines
	362: "ai", //Netherlands Antilles
	363: "aw", //Aruba
	364: "bs", //Bahamas (Commonwealth of the)
	365: "ai", //Anguilla
	366: "dm", //Dominica (Commonwealth of)
	368: "cu", //Cuba
	370: "do", //Dominican Republic
	372: "ht", //Haiti (Republic of)
	374: "tt", //Trinidad and Tobago
	376: "tc", //Turks and Caicos Islands
	400: "az", //Azerbaijani Republic
	401: "kz", //Kazakhstan (Republic of)
	402: "bt", //Bhutan (Kingdom of)
	404: "in", //India (Republic of)
	405: "in", //India (Republic of)
	406: "in", //India (Republic of)
	410: "pk", //Pakistan (Islamic Republic of)
	412: "af", //Afghanistan
	413: "lk", //Sri Lanka (Democratic Socialist Republic of)
	414: "mm", //Myanmar (Union of)
	415: "lb", //Lebanon
	416: "jo", //Jordan (Hashemite Kingdom of)
	417: "sy", //Syrian Arab Republic
	418: "iq", //Iraq (Republic of)
	419: "kw", //Kuwait (State of)
	420: "sa", //Saudi Arabia (Kingdom of)
	421: "ye", //Yemen (Republic of)
	422: "om", //Oman (Sultanate of)
	423: "ps", //Palestine
	424: "ae", //United Arab Emirates
	425: "il", //Israel (State of)
	426: "bh", //Bahrain (Kingdom of)
	427: "qa", //Qatar (State of)
	428: "mn", //Mongolia
	429: "np", //Nepal
	430: "ae", //United Arab Emirates
	431: "ae", //United Arab Emirates
	432: "ir", //Iran (Islamic Republic of)
	434: "uz", //Uzbekistan (Republic of)
	436: "tj", //Tajikistan (Republic of)
	437: "kg", //Kyrgyz Republic
	438: "tm", //Turkmenistan
	440: "jp", //Japan
	441: "jp", //Japan
	450: "kr", //Korea (Republic of)
	452: "vn", //Viet Nam (Socialist Republic of)
	454: "hk", //"Hong Kong, China"
	455: "mo", //"Macao, China"
	456: "kh", //Cambodia (Kingdom of)
	457: "la", //Lao People's Democratic Republic
	460: "cn", //China (People's Republic of)
	461: "cn", //China (People's Republic of)
	466: "tw", //"Taiwan, China"
	467: "kp", //Democratic People's Republic of Korea
	470: "bd", //Bangladesh (People's Republic of)
	472: "mv", //Maldives (Republic of)
	502: "my", //Malaysia
	505: "au", //Australia
	510: "id", //Indonesia (Republic of)
	514: "tl", //Democratic Republic of Timor-Leste
	515: "ph", //Philippines (Republic of the)
	520: "th", //Thailand
	525: "sg", //Singapore (Republic of)
	528: "bn", //Brunei Darussalam
	530: "nz", //New Zealand
	534: "mp", //Northern Mariana Islands (Commonwealth of the)
	535: "gu", //Guam
	536: "nr", //Nauru (Republic of)
	537: "pg", //Papua New Guinea
	539: "to", //Tonga (Kingdom of)
	540: "sb", //Solomon Islands
	541: "vu", //Vanuatu (Republic of)
	542: "fj", //Fiji (Republic of)
	543: "wf", //Wallis and Futuna (Territoire franais d'outre-mer)
	544: "as", //American Samoa
	545: "ki", //Kiribati (Republic of)
	546: "nc", //New Caledonia (Territoire franais d'outre-mer)
	547: "pf", //French Polynesia (Territoire franais d'outre-mer)
	548: "ck", //Cook Islands
	549: "ws", //Samoa (Independent State of)
	550: "fm", //Micronesia (Federated States of)
	551: "mh", //Marshall Islands (Republic of the)
	552: "pw", //Palau (Republic of)
	553: "tv", //Tuvalu
	555: "nu", //Niue
	602: "eg", //Egypt (Arab Republic of)
	603: "dz", //Algeria (People's Democratic Republic of)
	604: "ma", //Morocco (Kingdom of)
	605: "tn", //Tunisia
	606: "ly", //Libya (Socialist People's Libyan Arab Jamahiriya)
	607: "gm", //Gambia (Republic of the)
	608: "sn", //Senegal (Republic of)
	609: "mr", //Mauritania (Islamic Republic of)
	610: "ml", //Mali (Republic of)
	611: "gn", //Guinea (Republic of)
	612: "ci", //Côte d'Ivoire (Republic of)
	613: "bf", //Burkina Faso
	614: "ne", //Niger (Republic of the)
	615: "tg", //Togolese Republic
	616: "bj", //Benin (Republic of)
	617: "mu", //Mauritius (Republic of)
	618: "lr", //Liberia (Republic of)
	619: "sl", //Sierra Leone
	620: "gh", //Ghana
	621: "ng", //Nigeria (Federal Republic of)
	622: "td", //Chad (Republic of)
	623: "cf", //Central African Republic
	624: "cm", //Cameroon (Republic of)
	625: "cv", //Cape Verde (Republic of)
	626: "st", //Sao Tome and Principe (Democratic Republic of)
	627: "gq", //Equatorial Guinea (Republic of)
	628: "ga", //Gabonese Republic
	629: "cg", //Congo (Republic of the)
	630: "cg", //Democratic Republic of the Congo
	631: "ao", //Angola (Republic of)
	632: "gw", //Guinea-Bissau (Republic of)
	633: "sc", //Seychelles (Republic of)
	634: "sd", //Sudan (Republic of the)
	635: "rw", //Rwanda (Republic of)
	636: "et", //Ethiopia (Federal Democratic Republic of)
	637: "so", //Somali Democratic Republic
	638: "dj", //Djibouti (Republic of)
	639: "ke", //Kenya (Republic of)
	640: "tz", //Tanzania (United Republic of)
	641: "ug", //Uganda (Republic of)
	642: "bi", //Burundi (Republic of)
	643: "mz", //Mozambique (Republic of)
	645: "zm", //Zambia (Republic of)
	646: "mg", //Madagascar (Republic of)
	647: "re", //Reunion (French Department of)
	648: "zw", //Zimbabwe (Republic of)
	649: "na", //Namibia (Republic of)
	650: "mw", //Malawi
	651: "ls", //Lesotho (Kingdom of)
	652: "bw", //Botswana (Republic of)
	653: "sz", //Swaziland (Kingdom of)
	654: "km", //Comoros (Union of the)
	655: "za", //South Africa (Republic of)
	657: "er", //Eritrea
	658: "sh", //Saint Helena, Ascension and Tristan da Cunha
	659: "ss", //South Sudan (Republic of)
	702: "bz", //Belize
	704: "gt", //Guatemala (Republic of)
	706: "sv", //El Salvador (Republic of)
	708: "hn", //Honduras (Republic of)
	710: "ni", //Nicaragua
	712: "cr", //Costa Rica
	714: "pa", //Panama (Republic of)
	716: "pe", //Peru
	722: "ar", //Argentine Republic
	724: "br", //Brazil (Federative Republic of)
	730: "cl", //Chile
	732: "co", //Colombia (Republic of)
	734: "ve", //Venezuela (Bolivarian Republic of)
	736: "bo", //Bolivia (Republic of)
	738: "gy", //Guyana
	740: "ec", //Ecuador
	742: "gf", //French Guiana (French Department of)
	744: "py", //Paraguay (Republic of)
	746: "sr", //Suriname (Republic of)
	748: "uy", //Uruguay (Eastern Republic of)
	750: "fk", //Falkland Islands (Malvinas)
}
