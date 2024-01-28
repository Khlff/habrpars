FROM python:3.10

ENV DJANGO_ENV production

WORKDIR /djangoapp

COPY ./requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

CMD ["python", "manage.py", "runserver", "0.0.0.0:8000"]