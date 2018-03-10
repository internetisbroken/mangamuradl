// 180310
// 分割された画像のURLの配列の配列
// array[0][0]: 左上, array[0][1]: 最上段の左から2番目
// array[1][0]: 2段目の左端
// ...
return (function() {
	var a = document.querySelectorAll("img");
	var data = {};
	for(var i = 0; i < a.length; i++) {
		var element = a[i];
		var t = 0, left = 0;
		do {
			t += element.offsetTop  || 0;
			left += element.offsetLeft || 0;
			element = element.offsetParent;
		} while(element);
		if(! data[t]) {
			data[t] = {};
		}
		data[t][left] = a[i].src;
	}

	var comp = function(a, b) {
		return((a*1) - (b*1));
	}

	var k0 = [];
	for(var k in data) {
		k0.push(k);
	}
	k0 = k0.sort(comp);

	var line = [];
	for(var i = 0; i < k0.length; i++) {
		var key_0 = k0[i];
		var k1 = [];
		for(var t in data[key_0]) {
			k1.push(t);
		}
		k1 = k1.sort(comp);
		var urls = [];
		for(var j = 0; j < k1.length; j++) {
			var key_1 = k1[j];
			urls.push(data[key_0][key_1]);
		}
		line.push(urls);
	}

	return line;
})();
