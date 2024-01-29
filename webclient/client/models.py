from django.db import models


class Hub(models.Model):
    url = models.CharField(unique=True)
    timeout = models.PositiveIntegerField(default=600)

    class Meta:
        db_table = 'hubs'


class Article(models.Model):
    header = models.CharField(max_length=255)
    publication_date = models.DateField()
    url = models.TextField(unique=True)
    text = models.TextField()
    author_name = models.CharField(max_length=255)
    author_url = models.TextField()
    hub = models.ForeignKey(Hub, on_delete=models.CASCADE)

    class Meta:
        db_table = 'articles'

    def __str__(self):
        return self.header
