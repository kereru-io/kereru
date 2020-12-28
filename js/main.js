$(document).ready(function(){
	$("#pasteDateModal").on("shown.bs.modal", function(event){
		$('#pasteDateModalInput').focus();
	});
	$("#pasteDateModal").on("hidden.bs.modal", function(event){
		$('#pasteDateModalInput').val(null);
	});
});


$(document).ready(function(){
	$("#pasteDateModalSave").on("click", function(event){
		let value = moment.utc($('#pasteDateModalInput').val(), moment.HTML5_FMT.DATETIME_LOCAL_SECONDS);
		if (value.isValid()){
			$("#date").val(value.format(moment.HTML5_FMT.DATE));
			$("#time").val(value.format(moment.HTML5_FMT.TIME_SECONDS));
		}
	});
});


$(document).ready(function(){
	$(".mediapage").on("click", function(event){
		var PICKER = $("#mediapicker");
		var PAGE = PICKER.data("page");
		var OLDPAGE = PAGE;
		var TOTALPAGES = PICKER.data("totalpages");
		if (event.target.id == "nextpage"){
			if (PAGE != TOTALPAGES){
			PAGE=PAGE+1
			}
		}
		if (event.target.id == "backpage"){
			PAGE=PAGE-1
			if (PAGE == 0){PAGE=1}
		}
		if (event.target.id == "firstpage"){
			PAGE=1
		}
		if (event.target.id == "lastpage"){
			PAGE=TOTALPAGES
		}
		PICKER.data("page",PAGE);
		if (PAGE != OLDPAGE){
			DrawMediaPicker();
		}
	});
});


function MediaClicked(event){
	var IMAGE = $(this)
	var PICKER = $("#mediapicker");
	var ACTIVETAB = PICKER.data("tab");
	var ACTIVE = PICKER.data("selectid");
	$("#mediapicker img").removeClass("border-primary");
	if (IMAGE.data("imageid") == ACTIVE) {
		PICKER.data("selectid", "");
		PICKER.data("selecttype", "");
		PICKER.data("guid", "");
	} else {
		IMAGE.addClass("border-primary");
		PICKER.data("selectid", IMAGE.data("imageid"));
		PICKER.data("selecttype", ACTIVETAB);
		PICKER.data("guid", IMAGE.data("guid"));
	}
}


function MediaSave(){
	var MediaPicker = $("#mediapicker");
	var MediaSelected = MediaPicker.data("selectid");
	var MediaType = MediaPicker.data("selecttype");
	var GUID = MediaPicker.data("guid");
	$("#MediaID").val(MediaSelected);
	$("#MediaType").val(MediaType);
	if (GUID != "") {
		$("#mediathumb").attr("src",`/dashboard/media/view/thumbs/${GUID}`);
		$("#thumbview").removeClass("d-none");
		$("#addmediaview").addClass("d-none");
	}else{
		$("#thumbview").addClass("d-none");
		$("#addmediaview").removeClass("d-none");
	}
}


function MediaRemove(event){
	$("#mediathumb").removeAttr("src");
	var PICKER = $("#mediapicker");
	PICKER.data("selectid", "");
	PICKER.data("selecttype", "");
	PICKER.data("guid", "");
	$("#MediaID").val("");
	$("#MediaType").val("Image");
	$("#thumbview").addClass("d-none");
	$("#addmediaview").removeClass("d-none");
	event.stopPropagation();
}


$(document).ready(function(){
	$("#MediaRemove").on("click", MediaRemove );
	MediaSave();
});


$(document).ready(function(){
	$("#ImageModal").on("click",  function(event){
		var PICKER = $("#mediapicker");
		PICKER.data("tab", "Image");
		DrawMediaPicker();
	});
});


$(document).ready(function(){
	$("#VideoModal").on("click",  function(event){
		var PICKER = $("#mediapicker");
		PICKER.data("tab", "Video");
		DrawMediaPicker();
	});
});


$(document).ready(function(){
	$("#SaveModal").on("click", MediaSave );
	$("#addmedia").on("click", DrawMediaPicker );
	$(".card-img-overlay").on("click", DrawMediaPicker );
});


function DrawMediaPicker(){
	var PICKER = $("#mediapicker");
	var PAGE = PICKER.data("page");
	var ActiveMedia = PICKER.data("selectid");
	var ActiveMediaType = PICKER.data("selecttype");
	var MediaType = PICKER.data("tab");
	$( "#mediapicker img" ).remove();
	var boxes = [];
	$("#mediapicker div.col").each(function (i, el) {
		boxes.push($(el));
	});
	if (MediaType=="Image"){
		URL="/dashboard/images/list.json?page="+PAGE;
	}
	if (MediaType=="Video"){
		URL="/dashboard/videos/list.json?page="+PAGE;
	}
	$.getJSON( URL, function( data ) {
		$.each( data, function( key, val ) {
			if ((val.ID == ActiveMedia) && (MediaType == ActiveMediaType)){
				boxes[key].append(`<img class="border-primary img-thumbnail img-fluid" data-guid="${val.GUID}" data-imageid="${val.ID}" src="/dashboard/media/view/thumbs/${val.GUID}" alt="${val.DESC}" title="${val.DESC}">`);
			} else {
				boxes[key].append(`<img class="img-thumbnail img-fluid" data-guid="${val.GUID}" data-imageid="${val.ID}" src="/dashboard/media/view/thumbs/${val.GUID}" alt="${val.DESC}" title="${val.DESC}">`);
			}
		});
		$("#mediapicker img").on("click", MediaClicked );
	});
	$("#thispage").text(PAGE);
	$.getJSON( "/dashboard/images/pagecount.json", function( data ) {
		PICKER.data("totalpages", data.Totalpages);
	});
}


function CheckTweet(){
	var tweet = $("#tweet");
	if(!tweet.val()) {
		tweet.removeClass("is-valid").removeClass("is-invalid");
		tweet.siblings(".valid-feedback").text("");
		tweet.siblings(".invalid-feedback").text("");
		return;
	}
	var output = twttr.txt.parseTweet(tweet.val());
	if (output.valid){
		tweet.removeClass("is-invalid").addClass("is-valid");
		tweet.siblings(".valid-feedback").text(output.weightedLength);
	} else {
		tweet.removeClass("is-valid").addClass("is-invalid");
		tweet.siblings(".invalid-feedback").text(output.weightedLength);
	}
}


$(document).ready(function(){
	$("#tweet").ready( CheckTweet );
	$("#tweet").on("input", CheckTweet );
});
