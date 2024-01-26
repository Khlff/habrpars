from django.urls import path
from .views import hub_list, hub_edit, hub_add, hub_delete

urlpatterns = [
    path('hubs/', hub_list, name='hub_list'),
    path('hub/<int:pk>/', hub_edit, name='hub_edit'),
    path('hub/add/', hub_add, name='hub_add'),
    path('hub/<int:pk>/delete/', hub_delete, name='hub_delete'),
]
