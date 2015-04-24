var retryMills = 100
var loadCompleted = false
var refresh_screen_uri = "/api/screen/refresh_screen"
var isDialActivatedNow = false
var firstHandOnDial = true
var currentDialOrientation = 0.0 // 0~360 degrees
var previousDegree = 0.0

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





$(document).on("blur mouseup touchend", function(e) {
    console.log("you escapse! coming back on your browser");
    isDialActivatedNow=false;
});
$(document).on('touchmove', function(e) {
        if(e.target.id=="dial"){
            e.preventDefault();
        }
});
$(document).ready(function() {
    $("#dial").on("mousedown touchstart",function(e){
        console.log("we are in");
        isDialActivatedNow = true;
    });

    $("#dial").on('mousewheel wheel DOMMouseScroll', function(e){

        comparableWheeldata = 0;
        if(e.originalEvent.wheelDelta == undefined){
            comparableWheeldata = e.originalEvent.deltaY;
        }else{
            comparableWheeldata = e.originalEvent.wheelDelta * (-1);
        }
        if (comparableWheeldata>0){
            $.ajax({
                url: "/api/scpi/keyp",
                type: 'POST',
                data: {value: "DIAL:RIGH"},
                success: function (result){
                }
            });
            currentDialOrientation+=30
        }else{
            $.ajax({
                url: "/api/scpi/keyp",
                type: 'POST',
                data: {value: "DIAL:LEFT"},
                success: function (result){
                }
            });
            currentDialOrientation-=30
        }
        if(currentDialOrientation<0.0){
            currentDialOrientation += 360.0;
        }else if(currentDialOrientation>=360.0){
            currentDialOrientation -= 360.0;
        }
        //append new prefix
        current_dial_image_prefix = ((currentDialOrientation / 30)|0) + 1;
        console.log("prefix: " +  current_dial_image_prefix);
        imgpath = $("#dial").attr("src");
        firstpath = imgpath.split("_")[0];
        firstpath = firstpath + "_" + current_dial_image_prefix + ".png";
        $("#dial").attr("src", firstpath);
        e.preventDefault();
    });

    $("body").on("mousemove touchmove",function(e){
        if(isDialActivatedNow){
         var dialWidth = $("#dial").width();
         var dialHeight = $("#dial").height();
         var dialOffset = $("#dial").offset();
         var previous_dial_image_prefix = ((currentDialOrientation / 30)|0) + 1;
         dialOffset.top = dialOffset.top + dialHeight/2;
         dialOffset.left = dialOffset.left + dialWidth/2;
         // if its touch devices, pageX, pageY would not exist
         if(e.pageY == undefined || e.pageX == undefined){
             e.pageX = e.originalEvent.targetTouches[0].pageX;
             e.pageY = e.originalEvent.targetTouches[0].pageY;
         }
         var theta = Math.atan2(e.pageY-dialOffset.top, e.pageX-dialOffset.left) * 180 / Math.PI;
         theta += 90.0;
         if(theta < 0.0){
             theta += 360.0;
         }
         //console.log("Angle: " + theta);
         if(firstHandOnDial){
             firstHandOnDial=false;
         }else{
             currentDialOrientation+=theta-previousDegree;
             if(currentDialOrientation<0.0){
                 currentDialOrientation += 360.0;
             }else if(currentDialOrientation>360.0){
                 currentDialOrientation -= 360.0;
             }
         }
        previousDegree = theta;
        current_dial_image_prefix = ((currentDialOrientation / 30)|0) + 1;
        
        margin = current_dial_image_prefix-previous_dial_image_prefix;
        //1->12, 12->1
        margin_array = [current_dial_image_prefix, previous_dial_image_prefix];
        if($.inArray(12,margin_array)>-1 && $.inArray(1,margin_array)>-1){
               margin *= -1;
        }
        

        if(Math.abs(margin)>0){
            imgpath = $("#dial").attr("src");
            firstpath = imgpath.split("_")[0];
            firstpath = firstpath + "_" + current_dial_image_prefix + ".png";
            $("#dial").attr("src", firstpath);
        }
        if(margin>0){
			$.ajax({
				url: "/api/scpi/keyp",
				type: 'POST',
				data: {value: "DIAL:RIGH"},
				success: function (result){
				}
			});
            console.log("+1 : " + current_dial_image_prefix);
        }else if(margin<0){
			$.ajax({
				url: "/api/scpi/keyp",
				type: 'POST',
				data: {value: "DIAL:LEFT"},
				success: function (result){
				}
			});
            console.log("-1 : " + current_dial_image_prefix);
        }

        }
      });

      $("body").on("mouseup touchend",function(e){
          console.log("we are out");
          isDialActivatedNow = false;
      });

	  $(".cell").each(function(i){
		 $(this).click(function(e) {
			var offset = $(this).offset();
			x= e.pageX - offset.left
			y= e.pageY - offset.top
			x= parseInt(x*1.176)
			y= parseInt(y*1.176)
			$.ajax({
				url: "/api/scpi/touch",
				type: 'POST',
				data: {x: x, y: y},
				success: function (result){
				}
			});
		});
	 });
	 
	 $(".buttons").each(function(i){
         var button = $(this);
		 button.click(function(e) {
			$.ajax({
				url: "/api/scpi/keyp",
				type: 'POST',
				data: {value: $(this).attr("id")},
				success: function (result){
				}
			});
		});
                
        original_path = $(this).children("img").attr("src");
        button.children("img").attr("original_path", original_path);
        
        button.on("mousedown touchstart",function(e){
            pushed_button_image_path = $(this).children("img")
                                              .attr("original_path")
                                              .split(".png")
                                              .join("_d.png");
            $(this).children("img").attr("src", pushed_button_image_path);
        });
        button.on("mouseup touchend",function(e){
            original_button_image_path = $(this).children("img")
                                                .attr("original_path");
            $(this).children("img").attr("src", original_button_image_path);
        });
    });
	 
	 
	 ScreenAjaxRequest ();
});

function sendAPIAction(method, urls, values){
    $.ajax({
        url: urls,
        type: method,
        data: values,
        success: function (result){
        }
    });
}
