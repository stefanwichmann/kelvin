$(document).ready(function(){
  $('#dashboard').on('click', '.enableKelvinButton', function(){
    activateKelvin($(this).parents(".light"));
  });
  $('#dashboard').on('click', '#restartKelvinButton', function(){
    console.log("Restart kelvin button clicked");
    restartKelvin();
  });
});

function activateKelvin(entry) {
  console.log("Activating kelvin for light " + $(entry).attr("id"));
  $.ajax({
    url: "/lights/"+ $(entry).attr("id") +"/automatic",
    type: 'PUT'
  });
  $(entry).find(".enableKelvinButton").prop("disabled",true);
  window.setTimeout(function(){location.reload(true);}, 5000);
}

function restartKelvin() {
  $.ajax({
    url: "/restart",
    type: 'PUT'
  });
  $("#restartKelvinButton").prop("disabled",true);
  $("#message").append('<div class="alert alert-success alert-dismissable"><a href="#" class="close" data-dismiss="alert" aria-label="close">&times;</a><strong>Restarting... Please wait</div>');
  window.setTimeout(function(){location.reload(true);}, 5000);
}
