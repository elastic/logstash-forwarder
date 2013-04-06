#ifndef _ZMQ_COMPAT_H_
#define _ZMQ_COMPAT_H_

#  if ZMQ_VERSION_MAJOR == 2 /* zeromq 2 */
#    define zmq_compat_set_sendhwm(socket, hwm) zmq_setsockopt(socket, ZMQ_HWM, &hwm, sizeof(hwm))
#    define zmq_compat_set_recvhwm(socket, hwm) zmq_setsockopt(socket, ZMQ_HWM, &hwm, sizeof(hwm))
#    define zmq_compat_recvmsg(socket, message, flags) zmq_recv(socket, message, flags)
#    define zmq_compat_sendmsg(socket, message, flags) zmq_send(socket, message, flags)
#  elif ZMQ_VERSION_MAJOR == 3 /* zeromq 3 */
#    define zmq_compat_set_sendhwm(socket, hwm) zmq_setsockopt(socket, ZMQ_SNDHWM, &hwm, sizeof(hwm))
#    define zmq_compat_set_recvhwm(socket, hwm) zmq_setsockopt(socket, ZMQ_RCVHWM, &hwm, sizeof(hwm))
#    define zmq_compat_recvmsg(socket, message, flags) zmq_recvmsg(socket, message, flags)
#    define zmq_compat_sendmsg(socket, message, flags) zmq_sendmsg(socket, message, flags)
#  else
#    error "Unsupported zeromq version " ## ZMQ_VERSION_MAJOR
#  endif

#endif /* _ZMQ_COMPAT_H_ */
