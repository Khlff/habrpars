from django import forms
from .models import Hub


class HubForm(forms.ModelForm):
    class Meta:
        model = Hub
        fields = ['url', 'timeout']
