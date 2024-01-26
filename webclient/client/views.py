from django.shortcuts import render, get_object_or_404, redirect
from .models import Hub
from .forms import HubForm
import requests


def hub_list(request):
    hubs = Hub.objects.all()
    form = HubForm()
    return render(request, 'hub_list.html', {'hubs': hubs, 'form': form})


def hub_edit(request, pk):
    hub = get_object_or_404(Hub, pk=pk)
    if request.method == "POST":
        form = HubForm(request.POST, instance=hub)
        if form.is_valid():
            form.save()
            restart_parser()
            return redirect('hub_list')
    else:
        form = HubForm(instance=hub)
    return render(request, 'hub_edit.html', {'form': form})


def hub_add(request):
    if request.method == 'POST':
        form = HubForm(request.POST)
        if form.is_valid():
            form.save()
            restart_parser()
            return redirect('hub_list')
    else:
        form = HubForm()
    return render(request, 'hub_add.html', {'form': form})


def hub_delete(request, pk):
    hub = get_object_or_404(Hub, pk=pk)
    hub.delete()
    restart_parser()
    return redirect('hub_list')


def restart_parser():
    requests.get('http://localhost:8080/restart')  # restart go script
