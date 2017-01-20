FROM scratch
#MAINTAINER Máximo Cuadros <mcuadros@gmail.com>
ADD main /
CMD ["/main", "daemon", "--config", "/etc/ofelia/config.ini"]