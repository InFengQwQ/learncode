FROM python:3.11-alpine

RUN pip install --no-cache-dir "accesser[doh,doq]"

WORKDIR /accesser

EXPOSE 7654

CMD ["accesser", "--notsetproxy", "--notimportca"]
