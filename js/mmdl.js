// 180302 created
window.mmdl = {
	get_prepared: function() {
		if(! mmdl._prepared0) {
			if(typeof $ != "function" || typeof jQuery != "function") {
				mmdl.addjs("https://code.jquery.com/jquery-2.2.4.min.js");
				return false;
			} else if(typeof $.cookie != "function") {
				mmdl.addjs("https://cdnjs.cloudflare.com/ajax/libs/jquery-cookie/1.4.1/jquery.cookie.js");
			} else {
				mmdl._prepared0 = true;
			}
		}
		if(! mmdl._prepared1) {
			mmdl._prepared1 = true;
			mmdl.sio_addjs();
			mmdl.cap_addjs();
			mmdl.logwindow();
			return false;
		}
		if(typeof io != "function") {
			return false;
		}
		return true;
	},
	addjs: function(url) {
		if(! $('script[src="' + url + '"]').length) {
			$("head").append('<script src="'+url+'"></script>');
		}
	},
	///////////////////////////////////////////////////////
	// log
	///////////////////////////////////////////////////////
	logid: "log",
	logwindow: function(){
		$("body").prepend('<textarea id="' + mmdl.logid + '"></textarea>');
		$('#'+mmdl.logid).css({
			width: "60%",
			height: "50%",
			position: "fixed",
			top: "50%",
			left: "50%",
			transform: "translate(-50%, -50%)",
			"z-index" : "10000",
		});
		$('#'+mmdl.logid).css({cssText: $('#'+mmdl.logid).attr("style") + " !important;"});
	},
	log: function(msg) {
		$('#'+mmdl.logid).prepend(msg + "\n");
	},
	///////////////////////////////////////////////////////
	// dot org
	///////////////////////////////////////////////////////
	get_title: function() {
		var t =  $("section#data .title").text();
		return t == "" ? undefined : t;
	},
	get_list: function(id) {
		var data = $.ajax({
			type: 'GET',
			datatype: 'json',
			url: "/pages/xge6?id=" + id,
			async: false,
		});
		var t = data.responseText;
		if(t == "") {
			return "error: text is null";
		}
		return datas = $.parseJSON(t);
	},
	///////////////////////////////////////////////////////
	// recaptcha
	///////////////////////////////////////////////////////
	cap_addjs: function() {
		mmdl.addjs("https://www.google.com/recaptcha/api.js");
	},
	cap_cb: function(code){
		$.post("/cap/chkauth.php", { googleauth: code })
			.done(function(message) {
				mmdl.log("認証が完了しました");
				mmdl.cap_del();
			}).fail(function() {
			}).always(function() {
			});
	},
	cap_id: "cap",
	cap_running : function() {
		if($('#' + mmdl.cap_id).length) {
			return 1;
		}
		return 0;
	},
	cap_add: function(){
		mmdl.cap_del();
		mmdl.log("認証して下さい");
		$("body").prepend('<div id="'+mmdl.cap_id+'"></div>');
		$('#'+mmdl.cap_id).css({
			position: "fixed",
			top: "50%",
			left: "50%",
			transform: "translate(-50%, -50%)",
			"z-index" : "20000",
		});
		$('#'+mmdl.cap_id).css({cssText: $('#'+mmdl.cap_id).attr("style") + " !important;"});

		grecaptcha.render("cap", {
			sitekey: "6LdgkkYUAAAAAIWneJV4L3iK8C7e0x1Yl0CGhPCF",
			callback: "cap_cb",
		});
		window.cap_cb = mmdl.cap_cb;
	},
	cap_del: function(){
		$("#"+mmdl.cap_id).remove();
	},
	sio_addjs: function() {
		mmdl.addjs("https://cdnjs.cloudflare.com/ajax/libs/socket.io/2.0.4/socket.io.js");
	},
	///////////////////////////////////////////////////////
	// cookie
	///////////////////////////////////////////////////////
	cookie_name: "acookie4",
	cookie_update: function(cookie) {
		mmdl.log("updating cookie");
		$.cookie(mmdl.cookie_name, cookie, {
			expires: 1,
			path: "/",
		});
	},
	cookie_del: function() {
		mmdl.sio_disconnect();
		mmdl.log("deleting cookie");
		$.removeCookie(mmdl.cookie_name, "", {path: "/"});
	},
	cookie_load: function() {
		mmdl.cookie = mmdl.cookie_get();
	},
	cookie_get: function() {
		mmdl.log("getting cookie");
		return $.cookie(mmdl.cookie_name) ? $.cookie(mmdl.cookie_name) : "";
	},
	///////////////////////////////////////////////////////
	// socket.io
	///////////////////////////////////////////////////////
	sio_q_prefix: "page",
	sio_q_req: [],
	sio_q_pending: {},
	sio_q_img: [],
	sio_q_iframe: [],
	sio_q_blob: [],
	sio_getq_img: function() {
		var q = mmdl.sio_q_img;
		mmdl.sio_q_img = [];
		return q;
	},
	sio_getq_iframe: function() {
		var q = mmdl.sio_q_iframe;
		mmdl.sio_q_iframe = [];
		return q;
	},
	sio_getq_blob: function() {
		var q = mmdl.sio_q_blob;
		mmdl.sio_q_blob = [];
		return q;
	},
	sio_cb_img: function(data) {
		mmdl.sio_q_img.push({
			id: data.id,
			img: data.img,
		});
		delete mmdl.sio_q_pending[mmdl.sio_q_prefix+data.id];
		mmdl.log(data.id + ": " + data.img);
	},
	sio_cb_iframe: function(data) {
		mmdl.sio_q_iframe.push({
			id: data.id,
			img: data.img,
		});
		delete mmdl.sio_q_pending[mmdl.sio_q_prefix+data.id];
		mmdl.log(data.id + ": iframe " + data.img);
	},
	sio_cb_blob: function(data) {
		mmdl.sio_q_blob.push({
			id: data.id,
			b: data.b,
			img: data.img,
		});
		delete mmdl.sio_q_pending[mmdl.sio_q_prefix+data.id];
		mmdl.log(data.id + ": blob");
	},
	sio_cb_update: function(data) {
		if(data) {
			mmdl.cookie_update(data);
		}
	},
	sio_cb_del: function(data) {
		if(data) {
			mmdl.sio_disconnect();
			mmdl.cookie_del();
		}
	},
	sio_cb_auth: function(data) {
		if(data) {
			mmdl.sio_disconnect();
			mmdl.cookie_del();
			mmdl.cap_add();
		}
	},
	sio_connect: function() {
		if(! mmdl.get_prepared()) {
			return 0;
		}
		mmdl.sio_disconnect();
		mmdl.server = (function(){
			var min = 1;
			var max = 42;
			var sub = Math.floor(Math.random() * (max + 1 - min)) + min;
			return 'http://socket' + sub + '.spimg.ch';
		})();
		mmdl.socket = io(mmdl.server);
		mmdl.socket.on("return_img"   , mmdl.sio_cb_img   );
		mmdl.socket.on("return_iframe", mmdl.sio_cb_iframe);
		mmdl.socket.on("return_blob"  , mmdl.sio_cb_blob  );
		mmdl.socket.on("cookie_delete", mmdl.sio_cb_del   );
		mmdl.socket.on("cookie_update", mmdl.sio_cb_update);
		mmdl.socket.on("require_auth" , mmdl.sio_cb_auth  );

		mmdl.cookie_load();

		mmdl.socket.emit("first_connect", mmdl.cookie);
		mmdl.log("接続しました");

		return 1;
	},
	sio_disconnect: function() {
		if(typeof mmdl.socket == "object" && typeof mmdl.socket.close == "function") {
			if(mmdl.socket.connected) {
				mmdl.log("切断しました");
			}
			mmdl.socket.close();
		}
	},
	sio_request_add: function(id, url) {
		mmdl.sio_q_req.push({
			id: id,
			url: url,
		});
	},
	sio_pending2request: function() {
		var keys = Object.keys(mmdl.sio_q_pending);
		for(var i=0 ; i<keys.length; i++) {
			var k = keys[i];
			mmdl.sio_q_req.unshift(mmdl.sio_q_pending[k]);
			delete mmdl.sio_q_pending[k];
		}
	},
	sio_return: function(code) {
		return {
			code: code,
			n_request: mmdl.sio_q_req.length,
			n_pending: Object.keys(mmdl.sio_q_pending).length,
			img: mmdl.sio_getq_img(),
			iframe: mmdl.sio_getq_iframe(),
			blob: mmdl.sio_getq_blob(),
		};
	},
	sio_do: function(max) {
		if(! max) {
			max = 8;
		}

		if(mmdl.cap_running()) {
			return mmdl.sio_return(-2);
		}

		var cnt = 0;
		if(mmdl.sio_q_req.length > 0) {
			if(typeof mmdl.socket != "object" || mmdl.socket.connected == false) {
				mmdl.sio_connect();
				mmdl.sio_pending2request();
				return mmdl.sio_return(-1); // preparing
			}
			for( ; ; cnt++) {
				if(mmdl.sio_q_req.length <= 0 || cnt >= max) {
					break;
				}
				var data = mmdl.sio_q_req.shift();
				mmdl.socket.emit('request_img', {
					url        : data.url,
					cookie_data: mmdl.cookie,
					id         : data.id,
					viewr      : "o",
				});
				var k = mmdl.sio_q_prefix + data.id;
				mmdl.sio_q_pending[k] = data;
			}
		} else {
			if(typeof mmdl.socket != "object" || mmdl.socket.connected == false) {
				mmdl.sio_connect();
				mmdl.sio_pending2request();
				return mmdl.sio_return(-1); // preparing
			}
		}
		return mmdl.sio_return(cnt);
	},
};
