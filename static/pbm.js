/* pbm.js v0.0.4 Updated : 2018-01-23 */ ! function(e) {
    function t(r) {
        if (n[r]) return n[r].exports;
        var o = n[r] = {
            exports: {},
            id: r,
            loaded: !1
        };
        return e[r].call(o.exports, o, o.exports, t), o.loaded = !0, o.exports
    }
    var n = {};
    return t.m = e, t.c = n, t.p = "", t(0)
}([function(e, t, n) {
    n(1), e.exports = n(2)
}, function(e, t, n) {
    function r(e) {
        var t = localStorage.getItem(e);
        try {
            bidObj = JSON.parse(t)
        } catch (n) {
            return void u.logError("Issue parsing bid from localStorage :" + n.message)
        }
        s(bidObj)
    }

    function o(e) {
        var t = e,
            n = null,
            r = new XMLHttpRequest;
        return r.open("GET", "https://dev-prebid-cache.pub.network/cache?uuid=" + t, !1), r.withCredentials = "true", u.logTimestamp("Adm is requested"), r.send(null), 200 === r.status ? n = r.responseText : u.logError("Error request adm cache id"), n
    }

    function i(e, t, n) {
        var r = u.getUUID();
        return '<div id="' + r + '" style="border-style: none; position: absolute; width:100%; height:100%;">\n    <div id="' + r + '_inner" style="margin: 0 auto; width:' + t + "; height:" + n + '">' + e + "</div>\n    </div>"
    }

    function a(e) {
        var t;
        try {
            t = JSON.parse(e)
        } catch (n) {
            return void u.logError("Issue writing iframe into document :" + n.message)
        }
        s(t)
    }

    function s(e) {
        if (e) {
            if (e.error) return void u.logError("Issue writing iframe into document :" + e.error);
            var t, n;
            n = e.width ? e.width : e.w;
            var r;
            if (r = e.height ? e.height : e.h, e.adm) return t = i(e.adm, n, r), window.document.write(t), void(e.nurl && u.loadPixelUrl(c, e.nurl, u.getUUID()));
            if (e.nurl) {
                var o = u.loadScript(c, e.nurl);
                t = i(o.outerHTML, n, r), window.document.write(t)
            }
        }
    }
    var u = n(2),
        c = u.getWindow();
    c.pbm = {}, pbm.debug = pbm.debug || !1, pbm.enableDebug = function() {
        pbm.debug = !0, u.logInfo("Invoking pbm.enableDebug", arguments)
    }, pbm.disableDebug = function() {
        u.logInfo("Invoking pbm.disableDebug", arguments), pbm.debug = !1
    }, pbm.showAdFromCacheId = function(e) {
        u.logInfo("Invoking pbm.showAdFromCacheId", arguments);
        var t = e.admCacheID;
        if (t.startsWith("Prebid_")) r(t);
        else {
            var n = o(t);
            a(n)
        }
    }, window.apn_testonly = {};
    var d = window.apn_testonly;
    d.makeGetRequestForCachedBid = function(e) {
        return o(e)
    }, d.showAdFromResponse = function(e) {
        return a(e)
    }
}, function(e, t, n) {
    var r = null,
        o = n(3),
        i = o.TYPE.ARRAY,
        a = o.TYPE.STRING,
        s = o.TYPE.FUNC,
        u = o.TYPE.NUM,
        c = o.TYPE.OBJ,
        d = !1,
        l = o.DEBUG.DEBUG_MODE;
    try {
        r = "object" == typeof console.info ? console.info : console.info.bind(window.console)
    } catch (g) {}
    t.addEventHandler = function(e, t, n, r) {
        e.addEventListener ? e.addEventListener(t, n, r) : e.attachEvent && e.attachEvent("on" + t, n)
    }, t.removeEventHandler = function(e, t, n, r) {
        e.removeEventListener ? e.removeEventListener(t, n, r) : e.detachEvent && e.detachEvent("on" + t, n)
    }, t.isA = function(e, t) {
        return Object.prototype.toString.call(e) === "[object " + t + "]"
    }, t.isObj = function(e) {
        return this.isA(e, c)
    }, t.isFn = function(e) {
        return this.isA(e, s)
    }, t.isStr = function(e) {
        return this.isA(e, a)
    }, t.isArray = function(e) {
        return this.isA(e, i)
    }, t.isNumber = function(e) {
        return this.isA(e, u)
    }, t.isEmpty = function(e) {
        if (!e) return !0;
        if (this.isArray(e) || this.isStr(e)) return 0 === e.length;
        for (var t in e)
            if (hasOwnProperty.call(e, t)) return !1;
        return !0
    }, t.logMessage = function(e) {
        var t = h();
        this.debugTurnedOn() && f() && console.log(t + "MESSAGE" + e)
    }, t.logWarn = function(e) {
        var t = h();
        this.debugTurnedOn() && f() && (console.warn ? console.warn(t + "WARN: " + e) : console.log(t + "WARN: " + e))
    }, t.logError = function(e, t) {
        var n = t || "GENERAL_ERROR",
            r = h();
        this.debugTurnedOn() && f() && (console.error ? console.error(r + " " + n + ": " + e) : console.log(r + " " + n + ": " + e))
    }, t.logTimestamp = function(e) {
        this.debugTurnedOn() && f() && console.timeStamp && console.timeStamp(e)
    }, t.logInfo = function(e, t) {
        if (this.debugTurnedOn() && f()) {
            var n = h();
            r && (t && 0 !== t.length || (t = ""), r(n + "INFO: " + e + ("" === t ? "" : " : params : "), t))
        }
    }, t.loadScript = function(e, t, n) {
        var r = e.document,
            o = r.createElement("script");
        o.type = "text/javascript", n && "function" == typeof n && (o.readyState ? o.onreadystatechange = function() {
            "loaded" !== o.readyState && "complete" !== o.readyState || (o.onreadystatechange = null, n())
        } : o.onload = function() {
            n()
        }), o.src = t;
        var i = r.getElementsByTagName("head");
        return i = i.length ? i : r.getElementsByTagName("body"), i.length && (i = i[0], i.insertBefore(o, i.firstChild)), o
    }, t.getUUID = function() {
        var e = (new Date).getTime(),
            t = "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, function(t) {
                var n = (e + 16 * Math.random()) % 16 | 0;
                return e = Math.floor(e / 16), ("x" === t ? n : 3 & n | 8).toString(16)
            });
        return t
    }, t.loadPixelUrl = function(e, t, n) {
        var r, o = e.document.getElementsByTagName("head");
        if (e && o && t) {
            r = new Image, r.id = n, r.src = t, r.height = 0, r.width = 0, r.style.display = "none", r.onload = function() {
                try {
                    this.parentNode.removeChild(this)
                } catch (e) {}
            };
            try {
                o = o.length ? o : e.document.getElementsByTagName("body"), o.length && (o = o[0], o.insertBefore(r, o.firstChild))
            } catch (i) {
                this.logError("Error logging impression for tag: " + n + " :" + i.message)
            }
        }
    }, t._each = function(e, t) {
        if (!this.isEmpty(e)) {
            if (this.isFn(e.forEach)) return e.forEach(t);
            var n = 0,
                r = e.length;
            if (r > 0)
                for (; n < r; n++) t(e[n], n, e);
            else
                for (n in e) hasOwnProperty.call(e, n) && t(e[n], n, e)
        }
    }, t.contains = function(e, t) {
        if (this.isEmpty(e)) return !1;
        for (var n = e.length; n--;)
            if (e[n] === t) return !0;
        return !1
    };
    var f = function() {
        return window.console && window.console.log
    };
    t.debugTurnedOn = function() {
        return this.getWindow().pbm = this.getWindow().pbm || {}, pbm && pbm.debug === !1 && d === !1 && (pbm.debug = "TRUE" === this.getParameterByName(l).toUpperCase(), d = !0), !(!pbm || !pbm.debug)
    }, t.stringContains = function(e, t) {
        return !!e && e.indexOf(t) !== -1
    }, t.getSearchQuery = function() {
        try {
            return window.top.location.search
        } catch (e) {
            try {
                return window.location.search
            } catch (e) {
                return ""
            }
        }
    }, t.getParameterByName = function(e, t) {
        var n = "[\\?&]" + e + "=([^&#]*)",
            r = new RegExp(n),
            o = r.exec(t || this.getSearchQuery());
        return null === o ? "" : decodeURIComponent(o[1].replace(/\+/g, " "))
    }, t.hasOwn = function(e, t) {
        return e.hasOwnProperty ? e.hasOwnProperty(t) : typeof e[t] !== UNDEFINED && e.constructor.prototype[t] !== e[t]
    };
    var h = function() {
        var e = new Date,
            t = "[" + e.getHours() + ":" + e.getMinutes() + ":" + e.getSeconds() + ":" + e.getMilliseconds() + "] ";
        return t
    };
    t.getTargetArrayforRefresh = function(e) {
        var t = [];
        return this.isArray(e) ? t = e : this.isStr(e) && t.push(e), t
    }, t._map = function(e, t) {
        if (this.isEmpty(e)) return [];
        if (this.isFn(e.map)) return e.map(t);
        var n = [];
        return this._each(e, function(r, o) {
            n.push(t(r, o, e))
        }), n
    }, t.getValueString = function(e, t, n) {
        return void 0 === t || null === t ? n : this.isStr(t) ? t : this.isNumber(t) ? t.toString() : void this.logWarn("Unsuported type for param: " + e + " required type: String")
    }, t.getValueAsType = function(e, t, n, r) {
        return void 0 === t || null === t ? r : this.isA(t, n) ? t : (this.logWarn("Unsuported type for param: " + e + " required type: " + n), n === u && (t = Number(t)), isNaN(t) ? r : t)
    }, t.getWindow = function() {
        return window
    }, t.getAdObjFromAdsArray = function(e) {
        if (e && e.length > 0) {
            if (e[0][RTB]) return e[0][RTB];
            if (e[0][CSM]) return e[0][CSM];
            if (e[0][SSM]) return e[0][SSM]
        }
    }, t.cloneAsObject = function(e) {
        if (null === e || !(e instanceof Object)) return e;
        var t = e instanceof Array ? [] : {};
        for (var n in e) t[n] = this.cloneAsObject(e[n]);
        return t
    }
}, function(e, t) {
    e.exports = {
        LOG: {
            WARN: "WARN"
        },
        DEBUG: {
            DEBUG_MODE: "ast_debug"
        },
        OBJECT_TYPE: {
            UNDEFINED: "undefined",
            OBJECT: "object",
            STRING: "string",
            NUMBER: "number"
        },
        AD_TYPE: {
            BANNER: "banner",
            NATIVE: "native",
            VIDEO: "video"
        },
        TYPE: {
            ARRAY: "Array",
            STRING: "String",
            FUNC: "Function",
            NUM: "Number",
            OBJ: "Object",
            BOOL: "Boolean"
        },
        SAFEFRAME: {
            DEFAULT_ZINDEX: 3e3,
            STATUS: {
                READY: "ready"
            }
        }
    }
}]);