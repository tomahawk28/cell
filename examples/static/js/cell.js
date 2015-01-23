var retryMills = 100
var loadCompleted = false
var refresh_screen_uri = "/api/screen/refresh_screen"

//For handling IE Warning: 
if (typeof console === "undefined" || typeof console.log === "undefined") {
    console = {}
    console.log = function() {};
}

function checkLoadStatus(retry){
    if (loadCompleted){
        loadCompleted=false
        setTimeout(ScreenAjaxRequest,retryMills);
    }else if (retry>0){
       setTimeout(function() {
                checkLoadStatus(retry-1);
                }, 400);
       if(typeof(console)!=undefined){
           console.log("image load waiting: " + retry )
       }
    }else{
        //retry dried out
       if(typeof(console)!=undefined){
        console.log("image load faild: " + retry )
       }
        alert("Check your network connection!")
    }
    
}

function completeStatus(){
    loadCompleted = true
}

var ScreenAjaxRequest = function () {
	$(".cell").each(function(i){
		img = $(this)
		$.ajax({
			url: refresh_screen_uri,
			type: 'POST',
			timeout: 8000
		}).fail(function(jqXHR, textStatus, errorThrown){
			if(textStatus=="timeout"){
				$("#info").html("Poor network connection")
				setTimeout(ScreenAjaxRequest,retryMills*1.5);
			}else{
				alert("Connection failed, Try again:")
			}
		}).done(function(result){
			if (result != undefined){
                if(result.success){
                    $("#info").html("")
                    $(img).attr("src", "/api/screen/screen?date="+new Date().getTime());
                }else{
                    $("#info").html("Refresh Screenshot failed, " + result.data)
                }
			}
            checkLoadStatus(40)
		});
	});
};





$(document).ready(function() {
	  $(".cell").each(function(i){
		 $(this).click(function(e) {
			var offset = $(this).offset();
			x= e.pageX - offset.left
			y= e.pageY - offset.top
			x= parseInt(x*1.33)
			y= parseInt(y*1.33)
			$.ajax({
				url: "/api/scpi/touch",
				type: 'POST',
				data: {x: x, y: y},
				success: function (result){
				}
			});
		});
	 });
	 
	 $(".keyp").each(function(i){
		 $(this).click(function(e) {
			$.ajax({
				url: "/api/scpi/keyp",
				type: 'POST',
				data: {value: $(this).attr("value")},
				success: function (result){
				}
			});
		});
	 });
	 
	 ScreenAjaxRequest ();
});
