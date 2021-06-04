import os
import time
import logging
import grpc # Change to grpcio???
from grpc_interceptor import ServerInterceptor
import jwt

# Logger setup
try:
	logger = logging.getLogger(__file__.rsplit("/")[-3].rsplit(".")[0])
	logger.setLevel(logging.DEBUG)
except:
    print("Unable to initialise log file, good luck :)")

class AuthenticationInterceptor(ServerInterceptor):
	
	def __init__(self, secretKey, tokenDuration, authenticatedMethods):
		self.secretKey = secretKey
		self.tokenDuration = tokenDuration
		self.authenticatedMethods = authenticatedMethods
	
	def authorise(self, methodName, context):
    	# This function goes through a series of checks to verify that the user making a request is properly authenticated for that request

		# Check if the method requires authentication
		accessibleRoles = self.authenticatedMethods.get(methodName)
		if accessibleRoles == None:
			logger.info(f"Authentication is not required for {methodName}")
			return
			
		# Check if the request has metadata in it
		try:
			metadata = dict(context.invocation_metadata())
		except:
			logger.debug("Failed to authenticate: metadata is not provided")
			return grpc.StatusCode.PERMISSION_DENIED

		# Check if a JWT has been included in the metadata
		try:
			encodedToken = metadata["authorisation"]
		except:
			logger.debug("Failed to authenticate: JWT has not been provided")
			return grpc.StatusCode.PERMISSION_DENIED

		# Check that the provided JWT is valid
		claims, err = self.verifyJWT(encodedToken)
		if (err != None):
			logger.debug("Failed to authenticate: Provided JWT is invalid")
			return err

			#RESUME: HANDLE ERRORS BETTER (return claims, err) AND IMPLEMENT ROLE CHECK

		# Check that the role of the user making the service call authenticates them for the service being called
		for role in accessibleRoles:
			if (role == claims["role"]):
				logger.debug(f"Successfully authenticated request for {methodName}")
				return

		logger.debug("Failed to authenticate: the user does not have permission to access the requested service")
		return grpc.StatusCode.PERMISSION_DENIED

	def verifyJWT(self, accessToken):
		try:
			token = jwt.decode(accessToken, self.secretKey, algorithms=["HS256"])
		except Exception as e:
			logger.debug(f"Invalid token: {e}")
			return None, grpc.StatusCode.PERMISSION_DENIED
		
		return token, None

	def intercept(self, method, request, context, methodName):
		logger.info("Starting server-side authentication interceptor")

		# This isn't working but the error isn't being handled by the actual service
		err = self.authorise(methodName, context)
		if err:
			return err	
		
		return method(request, context)    