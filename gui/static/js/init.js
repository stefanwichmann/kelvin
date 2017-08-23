$(document).ready(function(){
  $(".continueButton").click(function(){
    console.log("Continue button clicked");
    uploadConfiguration($("#ip").val());
  });
});

function uploadConfiguration(ip) {
  var configuration = Object();
  var bridge = Object();
  bridge.IP = ip;
  configuration.bridge = bridge;
  console.log(JSON.stringify(configuration));
  $.ajax({
    url: "/configuration",
    type: 'PUT',
    data: JSON.stringify(configuration),
    contentType: 'application/json',
    success: function(result) {
      console.log(result);
      if (result == "success") {
        restartKelvin();
        window.setTimeout(function(){location.reload(true);}, 5000);
      } else {
        console.log(result);
      }
    }
  });
}

function restartKelvin() {
  $.ajax({
    url: "/restart",
    type: 'PUT'
  });
}
